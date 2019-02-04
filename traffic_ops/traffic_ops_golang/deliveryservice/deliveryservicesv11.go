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

type TODeliveryServiceV11 struct {
	ReqInfo *api.APIInfo
	tc.DeliveryServiceNullableV11
}

func V11ToLatest(ds *TODeliveryServiceV11) *TODeliveryService {
	latest := &TODeliveryService{ReqInfo: ds.ReqInfo}
	latest.DeliveryServiceNullableV11 = ds.DeliveryServiceNullableV11
	latest.Sanitize()
	return latest
}

func (v *TODeliveryServiceV11) APIInfo() *api.APIInfo { return v.ReqInfo }

func TypeFactoryV11(reqInfo *api.APIInfo) api.CRUDer {
	return &TODeliveryServiceV11{reqInfo, tc.DeliveryServiceNullableV11{}}
}

func (ds TODeliveryServiceV11) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return V11ToLatest(&ds).GetKeyFieldsInfo()
}

func (ds TODeliveryServiceV11) GetKeys() (map[string]interface{}, bool) {
	return V11ToLatest(&ds).GetKeys()
}

func (ds *TODeliveryServiceV11) SetKeys(keys map[string]interface{}) {
	latest := V11ToLatest(ds)
	latest.SetKeys(keys)
	ds.DeliveryServiceNullableV11 = latest.DeliveryServiceNullableV11 // TODO verify this is necessary
}

func (ds *TODeliveryServiceV11) GetAuditName() string {
	return V11ToLatest(ds).GetAuditName()
}

func (ds *TODeliveryServiceV11) GetType() string {
	return V11ToLatest(ds).GetType()
}

// IsTenantAuthorized checks that the user is authorized for both the delivery service's existing tenant, and the new tenant they're changing it to (if different).
func (ds *TODeliveryServiceV11) IsTenantAuthorized(user *auth.CurrentUser) (bool, error) {
	return V11ToLatest(ds).IsTenantAuthorized(user)
}

func (ds *TODeliveryServiceV11) Validate() error {
	return ds.DeliveryServiceNullableV11.Validate(ds.ReqInfo.Tx.Tx)
}

// Create is unimplemented, needed to satisfy CRUDer, since the framework doesn't allow a create to return an array of one
func (ds *TODeliveryServiceV11) Create() (error, error, int) {
	return nil, nil, http.StatusNotImplemented
}

func CreateV11(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()
	ds := tc.DeliveryServiceNullableV11{}
	if err := api.Parse(r.Body, inf.Tx.Tx, &ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}
	dsv13 := tc.NewDeliveryServiceNullableFromV11(ds)
	dsv13, errCode, userErr, sysErr = create(inf.Tx.Tx, *inf.Config, inf.User, dsv13)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.DeliveryServiceNullableV11{dsv13.DeliveryServiceNullableV11})
}

func (ds *TODeliveryServiceV11) Read() ([]interface{}, error, error, int) {
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
		returnable = append(returnable, ds.DeliveryServiceNullableV11)
	}
	return returnable, nil, nil, http.StatusOK
}

// Delete is the DeliveryService implementation of the Deleter interface
//all implementations of Deleter should use transactions and return the proper errorType
func (ds *TODeliveryServiceV11) Delete() (error, error, int) {
	return V11ToLatest(ds).Delete()
}

// Update is unimplemented, needed to satisfy CRUDer, since the framework doesn't allow an update to return an array of one
func (ds *TODeliveryServiceV11) Update() (error, error, int) {
	return nil, nil, http.StatusNotImplemented
}

func UpdateV11(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, []string{"id"}, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.DeliveryServiceNullableV11{}
	ds.ID = util.IntPtr(inf.IntParams["id"])
	if err := api.Parse(r.Body, inf.Tx.Tx, &ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}
	dsv13 := tc.NewDeliveryServiceNullableFromV11(ds)
	dsv13, errCode, userErr, sysErr = update(inf.Tx.Tx, *inf.Config, inf.User, &dsv13)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.DeliveryServiceNullableV11{dsv13.DeliveryServiceNullableV11})
}
