package servercapability

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
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/go-ozzo/ozzo-validation"
	"github.com/lib/pq"
)

type TOServerCapability struct {
	api.APIInfoImpl `json:"-"`
	tc.ServerCapability
}

func (v *TOServerCapability) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOServerCapability) NewReadObj() interface{}       { return &tc.ServerCapability{} }
func (v *TOServerCapability) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"id": dbhelpers.WhereColumnInfo{"server_id", api.IsInt},
	}
}

func (v *TOServerCapability) SelectQuery(where string, orderBy string, pagination string) string {
	return `
SELECT
  server_id,
  ARRAY_AGG(capability_name) as capability_names,
  MAX(last_updated) as last_updated
FROM
  server_capabilities
` + where + `
GROUP BY
  server_id
` + orderBy + pagination
}

type ServerCapabilityDB struct {
	tc.ServerCapability
	DBCapabilities pq.StringArray `db:"capability_names"`
}

func (scdb ServerCapabilityDB) Get() tc.ServerCapability {
	sc := scdb.ServerCapability
	sc.Capabilities = ([]string)(scdb.DBCapabilities)
	return sc
}

func (sc TOServerCapability) GetAuditName() string {
	if sc.ID != nil {
		return strconv.Itoa(*sc.ID)
	}
	return "unknown"
}

func (sc TOServerCapability) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

func (sc TOServerCapability) GetKeys() (map[string]interface{}, bool) {
	if sc.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *sc.ID}, true
}

func (sc *TOServerCapability) SetKeys(keys map[string]interface{}) {
	id, _ := keys["id"].(int) // default to 0 if it doesn't exist
	sc.ID = &id
}

func (sc TOServerCapability) GetType() string {
	return "servercapabilities"
}

func (sc TOServerCapability) Validate() error {
	errs := validation.Errors{
		"id": validation.Validate(sc.ID, validation.NotNil, validation.Required),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (sc *TOServerCapability) Read() ([]interface{}, error, error, int) {
	inf := sc.APIInfo()
	params := inf.Params

	queryParamsToSQLCols := sc.ParamColumns()

	where, orderBy, pagination, queryValues, errs := dbhelpers.BuildWhereAndOrderByAndPagination(params, queryParamsToSQLCols)
	if len(errs) > 0 {
		return nil, util.JoinErrs(errs), nil, http.StatusBadRequest
	}

	qry := sc.SelectQuery(where, orderBy, pagination)

	rows, err := inf.Tx.NamedQuery(qry, queryValues)
	if err != nil {
		userErr, sysErr, errCode := api.ParseDBError(err)
		return nil, userErr, sysErr, errCode
	}
	defer rows.Close()

	serverCapabilities := []interface{}{}
	for rows.Next() {
		c := ServerCapabilityDB{}
		if err := rows.StructScan(&c); err != nil {
			userErr, sysErr, errCode := api.ParseDBError(err)
			return nil, userErr, sysErr, errCode
		}
		serverCapabilities = append(serverCapabilities, c.Get())
	}

	return serverCapabilities, nil, nil, http.StatusOK
}

func (sc *TOServerCapability) Update() (error, error, int) {
	inf := sc.APIInfo()

	deleteQry := `DELETE FROM server_capabilities WHERE server_id=:server_id`
	if _, err := inf.Tx.NamedExec(deleteQry, sc); err != nil {
		return api.ParseDBError(err)
	}

	if len(sc.Capabilities) > 0 {
		insertQry := `
INSERT INTO server_capabilities(server_id, capability_name)
  SELECT $1, capabilities
  FROM UNNEST($2::text[]) capabilities
`
		if _, err := inf.Tx.Tx.Exec(insertQry, sc.ID, pq.Array(sc.Capabilities)); err != nil {
			return api.ParseDBError(err)
		}
	}
	api.CreateChangeLogRawTx(api.ApiChange, "ServerID: "+strconv.Itoa(*sc.ID)+" ACTION: Replace existing server capabilities assigned to server", inf.User, inf.Tx.Tx)
	return nil, nil, http.StatusOK
}
