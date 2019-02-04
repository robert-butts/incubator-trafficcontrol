package deliveryservice

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
	"errors"
	"net/http"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
)

type TODeliveryServiceV13 struct {
	ReqInfo *api.APIInfo
	tc.DeliveryServiceNullableV13
}

func V13ToLatest(ds *TODeliveryServiceV13) *TODeliveryService {
	latest := &TODeliveryService{ReqInfo: ds.ReqInfo}
	latest.DeliveryServiceNullableV13 = ds.DeliveryServiceNullableV13
	latest.Sanitize()
	return latest
}

func (v *TODeliveryServiceV13) APIInfo() *api.APIInfo { return v.ReqInfo }

func TypeFactoryV13(reqInfo *api.APIInfo) api.CRUDer {
	return &TODeliveryServiceV13{reqInfo, tc.DeliveryServiceNullableV13{}}
}

func (ds TODeliveryServiceV13) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return V13ToLatest(&ds).GetKeyFieldsInfo()
}

func (ds TODeliveryServiceV13) GetKeys() (map[string]interface{}, bool) {
	return V13ToLatest(&ds).GetKeys()
}

func (ds *TODeliveryServiceV13) SetKeys(keys map[string]interface{}) {
	latest := V13ToLatest(ds)
	latest.SetKeys(keys)
	ds.DeliveryServiceNullableV13 = latest.DeliveryServiceNullableV13 // TODO verify this is necessary
}

func (ds *TODeliveryServiceV13) GetAuditName() string {
	return V13ToLatest(ds).GetAuditName()
}

func (ds *TODeliveryServiceV13) GetType() string {
	return V13ToLatest(ds).GetType()
}

// IsTenantAuthorized checks that the user is authorized for both the delivery service's existing tenant, and the new tenant they're changing it to (if different).
func (ds *TODeliveryServiceV13) IsTenantAuthorized(user *auth.CurrentUser) (bool, error) {
	return V13ToLatest(ds).IsTenantAuthorized(user)
}

func (ds *TODeliveryServiceV13) Validate() error {
	return ds.DeliveryServiceNullableV13.Validate(ds.ReqInfo.Tx.Tx)
}

// Create is unimplemented, needed to satisfy CRUDer, since the framework doesn't allow a create to return an array of one
func (ds *TODeliveryServiceV13) Create() (error, error, int) {
	return nil, nil, http.StatusNotImplemented
}

func CreateV13(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	ds := tc.DeliveryServiceNullableV13{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}
	dsv13 := tc.NewDeliveryServiceNullableFromV13(ds)
	dsv13, errCode, userErr, sysErr = create(inf.Tx.Tx, *inf.Config, inf.User, dsv13)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceNullableV13{dsv13.DeliveryServiceNullableV13})
}

func (ds *TODeliveryServiceV13) Read() ([]interface{}, error, error, int) {
	returnable := []interface{}{}
	dses, errs, _ := read(ds.APIInfo().Params, ds.APIInfo().Tx, ds.APIInfo().User)
	if len(errs) > 0 {
		for _, err := range errs {
			if err.Error() == `id cannot parse to integer` {
				return nil, errors.New("Resource not found."), nil, http.StatusNotFound //matches perl response
			}
		}
		return nil, nil, errors.New("reading ds v12: " + util.JoinErrsStr(errs)), http.StatusInternalServerError
	}
	for _, ds := range dses {
		returnable = append(returnable, ds.DeliveryServiceNullableV13)
	}
	return returnable, nil, nil, http.StatusOK
}

// Delete is the DeliveryService implementation of the Deleter interface
//all implementations of Deleter should use transactions and return the proper errorType
func (ds *TODeliveryServiceV13) Delete() (error, error, int) {
	return V13ToLatest(ds).Delete()
}

// Update is unimplemented, needed to satisfy CRUDer, since the framework doesn't allow an update to return an array of one
func (ds *TODeliveryServiceV13) Update() (error, error, int) {
	return nil, nil, http.StatusNotImplemented
}

func UpdateV13(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceNullableV13{}
	ds.ID = util.IntPtr(inf.IntParams["id"])
	if err := api.Parse(r.Body, inf.Tx.Tx, &ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}
	dsv13 := tc.NewDeliveryServiceNullableFromV13(ds)
	dsv13, errCode, userErr, sysErr = update(inf.Tx.Tx, *inf.Config, inf.User, &dsv13)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceNullableV13{dsv13.DeliveryServiceNullableV13})
}
