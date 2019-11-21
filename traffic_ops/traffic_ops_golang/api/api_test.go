package api

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCamelCase(t *testing.T) {
	testStrings := []string{"hello_world", "trailing_underscore_", "w_h_a_t____"}
	expected := []string{"helloWorld", "trailingUnderscore", "wHAT"}
	for i, str := range testStrings {
		if toCamelCase(str) != expected[i] {
			t.Errorf("expected: %v error, actual: %v", expected[i], toCamelCase(str))
		}
	}
}

// TestRespWrittenAfterErrFails tests that a WriteResp called after HandleErr will not be written.
func TestRespWrittenAfterErrFails(t *testing.T) {
	w := &MockHTTPResponseWriter{}
	r := &http.Request{URL: &url.URL{}}
	tx := (*sql.Tx)(nil)

	expectedCode := http.StatusUnauthorized
	expectedUserErr := errors.New("user unauthorized")

	HandleErr(w, r, tx, expectedCode, expectedUserErr, nil)
	WriteResp(w, r, "should not be written")

	actualCode := w.Code
	statusVal := r.Context().Value(tc.StatusKey)
	statusInt, ok := statusVal.(int)
	if ok {
		actualCode = statusInt
	}

	if actualCode != expectedCode {
		t.Errorf("code expected: %+v, actual %+v", expectedCode, actualCode)
	}

	alerts := tc.Alerts{}
	if err := json.Unmarshal(w.Body, &alerts); err != nil {
		t.Fatalf("unmarshalling actual body: %v", err)
	}
	for _, alert := range alerts.Alerts {
		if string(alert.Level) != tc.ErrorLevel.String() {
			t.Errorf("alert level expected: '%s', actual: '%s'", tc.ErrorLevel.String(), alert.Level)
		}
	}
}

func TestWriteResp(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteResp(w, r, "foo")
	})
}

func TestWriteRespRaw(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteRespRaw(w, r, "foo")
	})
}

func TestWriteRespVals(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteRespVals(w, r, "foo", map[string]interface{}{"a": "b"})
	})
}

func TestRespWriter(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		RespWriter(w, r, nil)("foo", nil)
	})
}

func TestRespWriterVals(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		RespWriterVals(w, r, nil, map[string]interface{}{"a": "b"})("foo", nil)
	})
}

func TestWriteRespAlert(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteRespAlert(w, r, tc.ErrorLevel, "foo error")
	})
}

func TestWriteRespAlertObj(t *testing.T) {
	apiWriteTest(t, func(w http.ResponseWriter, r *http.Request) {
		WriteRespAlertObj(w, r, tc.ErrorLevel, "foo error", "bar")
	})
}

// apiWriteTest tests that an API write func succeeds and writes a body and a 200.
func apiWriteTest(t *testing.T, write func(w http.ResponseWriter, r *http.Request)) {
	w := &MockHTTPResponseWriter{}
	r := &http.Request{URL: &url.URL{}}

	write(w, r)

	if w.Code == 0 {
		w.Code = http.StatusOK // emulate behavior of w.Write
	}

	actualCode := w.Code
	statusVal := r.Context().Value(tc.StatusKey)
	statusInt, ok := statusVal.(int)
	if ok {
		actualCode = statusInt
	}

	expectedCode := http.StatusOK

	if actualCode != expectedCode {
		t.Errorf("code expected: %+v, actual %+v", expectedCode, actualCode)
	}

	if len(w.Body) == 0 {
		t.Errorf("body len expected: >0, actual 0")
	}
}

type MockHTTPResponseWriter struct {
	Code int
	Body []byte
}

func (i *MockHTTPResponseWriter) WriteHeader(rc int) {
	i.Code = rc
}

func (i *MockHTTPResponseWriter) Write(b []byte) (int, error) {
	i.Body = append(i.Body, b...)
	return len(b), nil
}

func (i *MockHTTPResponseWriter) Header() http.Header {
	return http.Header{}
}

func TestParseRestrictFKConstraint(t *testing.T) {
	var testCases = []struct {
		description        string
		storageError       pq.Error
		expectedReturnCode int
	}{
		{
			description: "FK Constraint Error",
			storageError: pq.Error{
				Message: "update or delete on table \"foo\" violates foreign key constraint \"fk_foo_bar\" on table \"bar\"",
			},
			expectedReturnCode: http.StatusBadRequest,
		},
		{
			description: "FK Constraint Error with underscores in table name",
			storageError: pq.Error{
				Message: "update or delete on table \"foo_ser\" violates foreign key constraint \"fk_foo_bar\" on table \"bar_cap\"",
			},
			expectedReturnCode: http.StatusBadRequest,
		},
		{
			description: "Non FK Constraint Error",
			storageError: pq.Error{
				Message: "connection error",
			},
			expectedReturnCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Log("Starting test scenario: ", tc.description)
			_, _, sc := parseRestrictFKConstraint(&tc.storageError)
			if sc != tc.expectedReturnCode {
				t.Errorf("code expected: %v, actual %v", tc.expectedReturnCode, sc)
			}
		})
	}
}

func TestChangeLogWritten(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://foo.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ChangeLogWritten(req) {
		t.Fatalf("expected new request ChangeLogWritten false, actual true")
	}
	SetChangeLogWritten(req)
	if !ChangeLogWritten(req) {
		t.Fatalf("expected request after SetChangeLogWritten to be ChangeLogWritten true, actual false")
	}
}

func TestCloseWithoutChangeLog(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()
	i := testIdentifier{}

	keys, _ := i.GetKeys()
	expectedMessage := strings.ToUpper(i.GetType()) + ": " + i.GetAuditName() + ", ID: " + strconv.Itoa(keys["id"].(int)) + ", ACTION: " + Created + " " + i.GetType() + ", keys: { id:" + strconv.Itoa(keys["id"].(int)) + " }"

	mock.ExpectBegin()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT").WithArgs(ApiChange, expectedMessage, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	// user := auth.CurrentUser{ID: 1}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://to.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	cfg.DBQueryTimeoutSeconds = 99
	ctx := req.Context()
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(42))
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{})
	req = req.WithContext(ctx)

	AddUserToReq(req, auth.CurrentUser{
		UserName:     "bill",
		ID:           42,
		PrivLevel:    0,
		TenantID:     0,
		Role:         0,
		Capabilities: pq.StringArray([]string{}),
	})

	req.Method = http.MethodGet
	inf, userErr, sysErr, errCode := NewInfo(req, nil, nil)
	if userErr != nil || sysErr != nil {
		t.Fatalf("NewInfo userErr '%v' sysErr '%v' code %v", userErr, sysErr, errCode)
	}
	if ChangeLogWritten(req) {
		t.Fatalf("expected new request ChangeLogWritten false, actual true")
	}
	inf.Close()
	if ChangeLogWritten(req) {
		t.Fatalf("expected safe request after Close() to be ChangeLogWritten false, actual true")
	}

	req.Method = http.MethodPost
	inf, userErr, sysErr, errCode = NewInfo(req, nil, nil)
	if userErr != nil || sysErr != nil {
		t.Fatalf("NewInfo userErr '%v' sysErr '%v' code %v", userErr, sysErr, errCode)
	}
	if ChangeLogWritten(req) {
		t.Fatalf("expected new request ChangeLogWritten false, actual true")
	}
	inf.Close()
	if !ChangeLogWritten(req) {
		t.Fatalf("expected unsafe request after Close() to be ChangeLogWritten true, actual false")
	}
}

func TestCloseOnErrorWithoutChangelog(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	mock.ExpectBegin()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://to.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	cfg.DBQueryTimeoutSeconds = 99
	ctx := req.Context()
	ctx = context.WithValue(ctx, DBContextKey, db)
	ctx = context.WithValue(ctx, ConfigContextKey, cfg)
	ctx = context.WithValue(ctx, ReqIDContextKey, uint64(42))
	ctx = context.WithValue(ctx, PathParamsKey, map[string]string{})
	req = req.WithContext(ctx)

	AddUserToReq(req, auth.CurrentUser{
		UserName:     "bill",
		ID:           42,
		PrivLevel:    0,
		TenantID:     0,
		Role:         0,
		Capabilities: pq.StringArray([]string{}),
	})

	req.Method = http.MethodPost
	inf, userErr, sysErr, errCode := NewInfo(req, nil, nil)
	if userErr != nil || sysErr != nil {
		t.Fatalf("NewInfo userErr '%v' sysErr '%v' code %v", userErr, sysErr, errCode)
	}
	if ChangeLogWritten(req) {
		t.Fatalf("expected new request ChangeLogWritten false, actual true")
	}

	w := MockResponseWriter{}

	HandleErr(w, req, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("something went wrong"))
	inf.Close()
	if !ChangeLogWritten(req) {
		t.Fatalf("expected request after HandleErr then Close() to be ChangeLogWritten true, actual false")
	}
}

type MockResponseWriter struct{}

func (m MockResponseWriter) Header() http.Header        { return http.Header{} }
func (m MockResponseWriter) Write([]byte) (int, error)  { return 0, nil }
func (m MockResponseWriter) WriteHeader(statusCode int) {}
