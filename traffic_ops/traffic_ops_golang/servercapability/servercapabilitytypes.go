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
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/go-ozzo/ozzo-validation"
)

//we need a type alias to define functions on
type TOServerCapabilityType struct {
	api.APIInfoImpl `json:"-"`
	tc.ServerCapabilityType
}

func (v *TOServerCapabilityType) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TOServerCapabilityType) InsertQuery() string           { return insertQuery() }
func (v *TOServerCapabilityType) NewReadObj() interface{}       { return &tc.ServerCapabilityType{} }
func (v *TOServerCapabilityType) SelectQuery() string           { return selectQuery() }

func (v *TOServerCapabilityType) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"name": dbhelpers.WhereColumnInfo{"name", nil},
	}
}

func (v *TOServerCapabilityType) DeleteQuery() string { return deleteQuery() }

func (v TOServerCapabilityType) GetAuditName() string {
	if v.Name != nil {
		return *v.Name
	}
	return "unknown"
}

func (v TOServerCapabilityType) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"name", api.GetStringKey}}
}

func (v TOServerCapabilityType) GetKeys() (map[string]interface{}, bool) {
	if v.Name == nil {
		return map[string]interface{}{"name": ""}, false
	}
	return map[string]interface{}{"name": *v.Name}, true
}

func (v *TOServerCapabilityType) SetKeys(keys map[string]interface{}) {
	i, _ := keys["name"].(string) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	v.Name = &i
}

func (v TOServerCapabilityType) GetType() string {
	return "servercapabilitytype"
}

func (v TOServerCapabilityType) Validate() error {
	errs := validation.Errors{
		"name": validation.Validate(v.Name, validation.NotNil, validation.Required),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (v *TOServerCapabilityType) Create() (error, error, int) {
	// NOTE server_capability_types don't have IDs, so the insert query returns 0,
	//      GenericCreate will call SetKeys, which  will override the name with "" since it wasn't returned,
	//      and then we have to re-set the name. It's a bit hacky, but it lets us use GenericCreate as-is.
	name := v.Name
	userErr, sysErr, errCode := api.GenericCreate(v)
	v.Name = name // re-set, because GenericCreate calls SetKeys without a name, which will override it.
	return userErr, sysErr, errCode
}

func (v *TOServerCapabilityType) Read() ([]interface{}, error, error, int) { return api.GenericRead(v) }

func (v *TOServerCapabilityType) Delete() (error, error, int) { return api.GenericDelete(v) }

func insertQuery() string {
	// NOTE this returns 0, because server_capability_types doesn't have an ID, which is expected by GenericCreate.
	return `INSERT INTO server_capability_types (name) VALUES (:name) RETURNING 0, last_updated`
}

func selectQuery() string { return `SELECT name, last_updated FROM server_capability_types` }

func deleteQuery() string { return `DELETE FROM server_capability_types WHERE name=:name` }
