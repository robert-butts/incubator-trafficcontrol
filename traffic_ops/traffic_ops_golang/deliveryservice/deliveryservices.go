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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/auth"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/tenant"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

//we need a type alias to define functions on

func NewTODeliveryService(version float64) *TODeliveryService {
	return &TODeliveryService{IDeliveryServiceNullable: tc.NewDeliveryServiceNullable(version)}
}

type TODeliveryService struct {
	api.APIInfoImpl `json:"-"`
	tc.IDeliveryServiceNullable
}

func (ds *TODeliveryService) APIInfo() *api.APIInfo { return ds.ReqInfo }

func (ds *TODeliveryService) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	ds.SetID(&i)
}

// GetType implements api.GenericDeleter.
func (ds *TODeliveryService) GetType() string { return "ds" }

// GetType implements api.GenericDeleter.
func (ds *TODeliveryService) GetAuditName() string {
	latestDS := ds.ToLatest()
	if latestDS.XMLID != nil {
		return *latestDS.XMLID
	}
	return ""
}

func (ds TODeliveryService) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

func (ds TODeliveryService) GetKeys() (map[string]interface{}, bool) {
	latestDS := ds.ToLatest()
	if latestDS.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *latestDS.ID}, true
}

// DeleteQuery implements api.GenericDeleter.
func (ds *TODeliveryService) DeleteQuery() string {
	return `DELETE FROM ` + ds.DBTable() + ` WHERE id = :id`
}

func (ds *TODeliveryService) Validate() error {
	return ds.IDeliveryServiceNullable.Validate(ds.APIInfo().Tx.Tx)
}

// 	TODO allow users to post names (type, cdn, etc) and get the IDs from the names. This isn't trivial to do in a single query, without dynamically building the entire insert query, and ideally inserting would be one query. But it'd be much more convenient for users. Alternatively, remove IDs from the database entirely and use real candidate keys.
func Create(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, nil)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	ds := tc.NewDeliveryServiceNullable(inf.APIVersion)
	if err := api.Parse(r.Body, inf.Tx.Tx, ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("decoding: "+err.Error()), nil)
		return
	}
	latestDS, errCode, userErr, sysErr := create(inf, ds.ToLatest())
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice creation was successful.", []tc.IDeliveryServiceNullable{latestDS.ToVersion(inf.APIVersion)})
}

// create creates the given ds in the database, and returns the DS with its id and other fields created on insert set. On error, the HTTP status code, user error, and system error are returned. The status code SHOULD NOT be used, if both errors are nil.
func create(inf *api.APIInfo, ds *tc.DeliveryServiceNullable) (tc.DeliveryServiceNullable, int, error, error) {
	user := inf.User
	tx := inf.Tx.Tx
	txx := inf.Tx
	cfg := inf.Config

	if authorized, err := isTenantAuthorized(inf, ds); err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("checking tenant: " + err.Error())
	} else if !authorized {
		return tc.DeliveryServiceNullable{}, http.StatusForbidden, errors.New("not authorized on this tenant"), nil
	}

	dsCopy := *ds
	qry := InsertQuery(&dsCopy)

	rows, err := txx.NamedQuery(qry, ds)
	if err != nil {
		usrErr, sysErr, code := api.ParseDBError(err)
		return tc.DeliveryServiceNullable{}, code, usrErr, sysErr
	}
	defer rows.Close()

	id := 0
	lastUpdated := tc.TimeNoMod{}
	if !rows.Next() {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("no deliveryservice request inserted, no id was returned")
	}
	if err := rows.Scan(&id, &lastUpdated); err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("could not scan id from insert: " + err.Error())
	}
	if rows.Next() {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("too many ids returned from deliveryservice request insert")
	}
	ds.ID = &id

	if ds.ID == nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("missing id after insert")
	}
	if ds.XMLID == nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("missing xml_id after insert")
	}
	if ds.TypeID == nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("missing type after insert")
	}
	dsType, err := getTypeFromID(*ds.TypeID, tx)
	if err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("getting delivery service type: " + err.Error())
	}
	ds.Type = &dsType

	if err := createDefaultRegex(tx, *ds.ID, *ds.XMLID); err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("creating default regex: " + err.Error())
	}

	matchlists, err := GetDeliveryServicesMatchLists([]string{*ds.XMLID}, tx)
	if err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("creating DS: reading matchlists: " + err.Error())
	}
	if matchlist, ok := matchlists[*ds.XMLID]; !ok {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("creating DS: reading matchlists: not found")
	} else {
		ds.MatchList = &matchlist
	}

	cdnName, cdnDomain, dnssecEnabled, err := getCDNNameDomainDNSSecEnabled(*ds.ID, tx)
	if err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("creating DS: getting CDN info: " + err.Error())
	}

	ds.ExampleURLs = MakeExampleURLs(ds.Protocol, *ds.Type, *ds.RoutingName, *ds.MatchList, cdnDomain)

	if err := EnsureParams(tx, *ds.ID, *ds.XMLID, ds.EdgeHeaderRewrite, ds.MidHeaderRewrite, ds.RegexRemap, ds.CacheURL, ds.SigningAlgorithm, dsType, ds.MaxOriginConnections); err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("ensuring ds parameters:: " + err.Error())
	}

	if dnssecEnabled {
		if err := PutDNSSecKeys(tx, cfg, *ds.XMLID, cdnName, ds.ExampleURLs); err != nil {
			return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("creating DNSSEC keys: " + err.Error())
		}
	}

	if err := createPrimaryOrigin(tx, user, *ds); err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("creating delivery service: " + err.Error())
	}

	ds.LastUpdated = &lastUpdated
	if err := api.CreateChangeLogRawErr(api.ApiChange, "Created ds: "+*ds.XMLID+" id: "+strconv.Itoa(*ds.ID), user, tx); err != nil {
		return tc.DeliveryServiceNullable{}, http.StatusInternalServerError, nil, errors.New("error writing to audit log: " + err.Error())
	}
	return *ds, http.StatusOK, nil, nil
}

func (ds *TODeliveryService) Read() ([]interface{}, error, error, int) {
	inf := ds.APIInfo()
	returnable := []interface{}{}
	dses, errs, _ := read(ds.APIInfo().Params, ds.APIInfo().Tx, ds.APIInfo().User)
	if len(errs) > 0 {
		for _, err := range errs {
			if err.Error() == `id cannot parse to integer` { // TODO create const for string
				return nil, errors.New("Resource not found."), nil, http.StatusNotFound //matches perl response
			}
		}
		return nil, nil, errors.New("reading dses: " + util.JoinErrsStr(errs)), http.StatusInternalServerError
	}

	for _, ds := range dses {
		returnable = append(returnable, ds.ToVersion(inf.APIVersion))
	}
	return returnable, nil, nil, http.StatusOK
}

func Update(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	ds := tc.NewDeliveryServiceNullable(inf.APIVersion)
	if err := json.NewDecoder(r.Body).Decode(&ds); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("malformed JSON: "+err.Error()), nil)
		return
	}
	ds.SetID(&id)

	if err := ds.Validate(inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusBadRequest, errors.New("invalid request: "+err.Error()), nil)
		return
	}

	latestDS, errCode, userErr, sysErr := update(inf, ds)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	api.WriteRespAlertObj(w, r, tc.SuccessLevel, "Deliveryservice update was successful.", []tc.IDeliveryServiceNullable{latestDS.ToVersion(inf.APIVersion)})
}

func Delete(w http.ResponseWriter, r *http.Request) {
	inf, userErr, sysErr, errCode := api.NewInfo(r, nil, []string{"id"})
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}
	defer inf.Close()

	id := inf.IntParams["id"]

	xmlID, ok, err := GetXMLID(inf.Tx.Tx, id)
	if err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("ds delete: getting xmlid: "+err.Error()))
	} else if !ok {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusNotFound, errors.New("delivery service not found"), nil)
	}

	// Note ds regexes MUST be deleted before the ds, because there's a ON DELETE CASCADE on deliveryservice_regex (but not on regex).
	// Likewise, it MUST happen in a transaction with the later DS delete, so they aren't deleted if the DS delete fails.
	if _, err := inf.Tx.Tx.Exec(`DELETE FROM regex WHERE id IN (SELECT regex FROM deliveryservice_regex WHERE deliveryservice=$1)`, id); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.Delete deleting regexes for delivery service: "+err.Error()))
		return
	}

	if _, err := inf.Tx.Tx.Exec(`DELETE FROM deliveryservice_regex WHERE deliveryservice=$1`, id); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.Delete deleting delivery service regexes: "+err.Error()))
		return
	}

	ds := NewTODeliveryService(inf.APIVersion)
	ds.SetInfo(inf)
	type DSID struct {
		TODeliveryService
		ID int `db:"id"`
	}
	dsID := &DSID{TODeliveryService: *ds, ID: id}
	userErr, sysErr, errCode = api.GenericDelete(dsID)
	if userErr != nil || sysErr != nil {
		api.HandleErr(w, r, inf.Tx.Tx, errCode, userErr, sysErr)
		return
	}

	paramConfigFilePrefixes := []string{"hdr_rw_", "hdr_rw_mid_", "regex_remap_", "cacheurl_"}
	configFiles := []string{}
	for _, prefix := range paramConfigFilePrefixes {
		configFiles = append(configFiles, prefix+xmlID+".config")
	}

	if _, err := inf.Tx.Tx.Exec(`DELETE FROM parameter WHERE name = 'location' AND config_file = ANY($1)`, pq.Array(configFiles)); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("deliveryservice.Delete deleting delivery service parameters: "+err.Error()))
		return
	}

	if err := api.CreateChangeLog(api.ApiChange, api.Deleted, ds, inf.User, inf.Tx.Tx); err != nil {
		api.HandleErr(w, r, inf.Tx.Tx, http.StatusInternalServerError, nil, errors.New("inserting changelog: "+err.Error()))
		return
	}
	api.WriteRespAlert(w, r, tc.SuccessLevel, ds.GetType()+" was deleted.")
}

func createDefaultRegex(tx *sql.Tx, dsID int, xmlID string) error {
	regexStr := `.*\.` + xmlID + `\..*`
	regexID := 0
	if err := tx.QueryRow(`INSERT INTO regex (type, pattern) VALUES ((select id from type where name = 'HOST_REGEXP'), $1::text) RETURNING id`, regexStr).Scan(&regexID); err != nil {
		return errors.New("insert regex: " + err.Error())
	}
	if _, err := tx.Exec(`INSERT INTO deliveryservice_regex (deliveryservice, regex, set_number) VALUES ($1::bigint, $2::bigint, 0)`, dsID, regexID); err != nil {
		return errors.New("executing parameter query to insert location: " + err.Error())
	}
	return nil
}

func update(inf *api.APIInfo, ds tc.IDeliveryServiceNullable) (*tc.DeliveryServiceNullable, int, error, error) {
	tx := inf.Tx.Tx
	txx := inf.Tx
	cfg := inf.Config
	user := inf.User

	latestDS := ds.ToLatest()

	if authorized, err := isTenantAuthorized(inf, latestDS); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("checking tenant: " + err.Error())
	} else if !authorized {
		return nil, http.StatusForbidden, errors.New("not authorized on this tenant"), nil
	}

	if latestDS.XMLID == nil {
		return nil, http.StatusBadRequest, errors.New("missing xml_id"), nil
	}
	if latestDS.ID == nil {
		return nil, http.StatusBadRequest, errors.New("missing id"), nil
	}

	dsType, ok, err := getDSType(tx, *latestDS.XMLID)
	if !ok {
		return nil, http.StatusNotFound, errors.New("delivery service '" + *latestDS.XMLID + "' not found"), nil
	}
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting delivery service type during update: " + err.Error())
	}

	// oldHostName will be used to determine if SSL Keys need updating - this will be empty if the DS doesn't have SSL keys, because DS types without SSL keys may not have regexes, and thus will fail to get a host name.
	oldHostName := ""
	if dsType.HasSSLKeys() {
		oldHostName, err = getOldHostName(*latestDS.ID, tx)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("getting existing delivery service hostname: " + err.Error())
		}
	}

	qry := UpdateQuery(ds)

	rows, err := txx.NamedQuery(qry, latestDS)

	if err != nil {
		usrErr, sysErr, code := api.ParseDBError(err)
		return nil, code, usrErr, sysErr
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, http.StatusNotFound, errors.New("no delivery service found with this id"), nil
	}
	lastUpdated := tc.TimeNoMod{}
	if err := rows.Scan(&lastUpdated); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("scan updating delivery service: " + err.Error())
	}
	if rows.Next() {
		xmlID := ""
		if latestDS.XMLID != nil {
			xmlID = *latestDS.XMLID
		}
		return nil, http.StatusInternalServerError, nil, errors.New("updating delivery service " + xmlID + ": " + "this update affected too many rows: > 1")
	}

	latestDS = ds.ToLatest()

	if latestDS.ID == nil {
		return nil, http.StatusInternalServerError, nil, errors.New("missing id after update")
	}
	if latestDS.XMLID == nil {
		return nil, http.StatusInternalServerError, nil, errors.New("missing xml_id after update")
	}
	if latestDS.TypeID == nil {
		return nil, http.StatusInternalServerError, nil, errors.New("missing type after update")
	}
	if latestDS.RoutingName == nil {
		return nil, http.StatusInternalServerError, nil, errors.New("missing routing name after update")
	}
	newDSType, err := getTypeFromID(*latestDS.TypeID, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting delivery service type after update: " + err.Error())
	}
	latestDS.Type = &newDSType

	cdnDomain, err := getCDNDomain(*latestDS.ID, tx) // need to get the domain again, in case it changed.
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting CDN domain after update: " + err.Error())
	}

	matchLists, err := GetDeliveryServicesMatchLists([]string{*latestDS.XMLID}, tx)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("getting matchlists after update: " + err.Error())
	}
	if ml, ok := matchLists[*latestDS.XMLID]; !ok {
		return nil, http.StatusInternalServerError, nil, errors.New("no matchlists after update")
	} else {
		latestDS.MatchList = &ml
	}

	// newHostName will be used to determine if SSL Keys need updating - this will be empty if the DS doesn't have SSL keys, because DS types without SSL keys may not have regexes, and thus will fail to get a host name.
	newHostName := ""
	if dsType.HasSSLKeys() {
		newHostName, err = getHostName(latestDS.Protocol, *latestDS.Type, *latestDS.RoutingName, *latestDS.MatchList, cdnDomain)
		if err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("getting hostname after update: " + err.Error())
		}
	}

	if newDSType.HasSSLKeys() && oldHostName != newHostName {
		if err := updateSSLKeys(latestDS, newHostName, tx, cfg); err != nil {
			return nil, http.StatusInternalServerError, nil, errors.New("updating delivery service " + *latestDS.XMLID + ": updating SSL keys: " + err.Error())
		}
	}

	if err := EnsureParams(tx, *latestDS.ID, *latestDS.XMLID, latestDS.EdgeHeaderRewrite, latestDS.MidHeaderRewrite, latestDS.RegexRemap, latestDS.CacheURL, latestDS.SigningAlgorithm, newDSType, latestDS.MaxOriginConnections); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("ensuring ds parameters:: " + err.Error())
	}

	if err := updatePrimaryOrigin(tx, user, *latestDS); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("updating delivery service: " + err.Error())
	}

	latestDS.LastUpdated = &lastUpdated

	if err := api.CreateChangeLogRawErr(api.ApiChange, "Updated ds: "+*latestDS.XMLID+" id: "+strconv.Itoa(*latestDS.ID), user, tx); err != nil {
		return nil, http.StatusInternalServerError, nil, errors.New("writing change log entry: " + err.Error())
	}
	return latestDS, http.StatusOK, nil, nil
}

func read(params map[string]string, tx *sqlx.Tx, user *auth.CurrentUser) ([]tc.DeliveryServiceNullable, []error, tc.ApiErrorType) {
	if strings.HasSuffix(params["id"], ".json") {
		params["id"] = params["id"][:len(params["id"])-len(".json")]
	}
	if _, ok := params["orderby"]; !ok {
		params["orderby"] = "xml_id"
	}

	// Query Parameters to Database Query column mappings
	// see the fields mapped in the SQL query
	queryParamsToSQLCols := map[string]dbhelpers.WhereColumnInfo{
		"id":               dbhelpers.WhereColumnInfo{"ds.id", api.IsInt},
		"cdn":              dbhelpers.WhereColumnInfo{"ds.cdn_id", api.IsInt},
		"xml_id":           dbhelpers.WhereColumnInfo{"ds.xml_id", nil},
		"xmlId":            dbhelpers.WhereColumnInfo{"ds.xml_id", nil},
		"profile":          dbhelpers.WhereColumnInfo{"ds.profile", api.IsInt},
		"type":             dbhelpers.WhereColumnInfo{"ds.type", api.IsInt},
		"logsEnabled":      dbhelpers.WhereColumnInfo{"ds.logs_enabled", api.IsBool},
		"tenant":           dbhelpers.WhereColumnInfo{"ds.tenant_id", api.IsInt},
		"signingAlgorithm": dbhelpers.WhereColumnInfo{"ds.signing_algorithm", nil},
	}

	where, orderBy, queryValues, errs := dbhelpers.BuildWhereAndOrderBy(params, queryParamsToSQLCols)
	if len(errs) > 0 {
		return nil, errs, tc.DataConflictError
	}

	tenantIDs, err := tenant.GetUserTenantIDListTx(tx.Tx, user.TenantID)

	if err != nil {
		log.Errorln("received error querying for user's tenants: " + err.Error())
		return nil, []error{tc.DBError}, tc.SystemError
	}

	where, queryValues = dbhelpers.AddTenancyCheck(where, queryValues, "ds.tenant_id", tenantIDs)

	query := SelectQuery(&tc.DeliveryServiceNullable{}) + where + orderBy

	return GetDeliveryServices(query, queryValues, tx)
}

func getOldHostName(id int, tx *sql.Tx) (string, error) {
	q := `
SELECT ds.xml_id, ds.protocol, type.name, ds.routing_name, cdn.domain_name
FROM  deliveryservice as ds
JOIN type ON ds.type = type.id
JOIN cdn ON ds.cdn_id = cdn.id
WHERE ds.id=$1
`
	xmlID := ""
	protocol := (*int)(nil)
	dsTypeStr := ""
	routingName := ""
	cdnDomain := ""
	if err := tx.QueryRow(q, id).Scan(&xmlID, &protocol, &dsTypeStr, &routingName, &cdnDomain); err != nil {
		return "", fmt.Errorf("querying delivery service %v host name: "+err.Error()+"\n", id)
	}
	dsType := tc.DSTypeFromString(dsTypeStr)
	if dsType == tc.DSTypeInvalid {
		return "", errors.New("getting delivery services matchlist: got invalid delivery service type '" + dsTypeStr + "'")
	}
	matchLists, err := GetDeliveryServicesMatchLists([]string{xmlID}, tx)
	if err != nil {
		return "", errors.New("getting delivery services matchlist: " + err.Error())
	}
	matchList, ok := matchLists[xmlID]
	if !ok {
		return "", errors.New("delivery service has no match lists (is your delivery service missing regexes?)")
	}
	host, err := getHostName(protocol, dsType, routingName, matchList, cdnDomain) // protocol defaults to 0: doesn't need to check Valid()
	if err != nil {
		return "", errors.New("getting hostname: " + err.Error())
	}
	return host, nil
}

func getTypeFromID(id int, tx *sql.Tx) (tc.DSType, error) {
	// TODO combine with getOldHostName, to only make one query?
	name := ""
	if err := tx.QueryRow(`SELECT name FROM type WHERE id = $1`, id).Scan(&name); err != nil {
		return "", fmt.Errorf("querying type ID %v: "+err.Error()+"\n", id)
	}
	return tc.DSTypeFromString(name), nil
}

func updatePrimaryOrigin(tx *sql.Tx, user *auth.CurrentUser, ds tc.DeliveryServiceNullable) error {
	count := 0
	q := `SELECT count(*) FROM origin WHERE deliveryservice = $1 AND is_primary`
	if err := tx.QueryRow(q, *ds.ID).Scan(&count); err != nil {
		return fmt.Errorf("querying existing primary origin for ds %s: %s", *ds.XMLID, err.Error())
	}

	if ds.OrgServerFQDN == nil || *ds.OrgServerFQDN == "" {
		if count == 1 {
			// the update is removing the existing orgServerFQDN, so the existing row needs to be deleted
			q = `DELETE FROM origin WHERE deliveryservice = $1 AND is_primary`
			if _, err := tx.Exec(q, *ds.ID); err != nil {
				return fmt.Errorf("deleting primary origin for ds %s: %s", *ds.XMLID, err.Error())
			}
			api.CreateChangeLogRawTx(api.ApiChange, "Deleted primary origin for delivery service: "+*ds.XMLID, user, tx)
		}
		return nil
	}

	if count == 0 {
		// orgServerFQDN is going from null to not null, so the primary origin needs to be created
		return createPrimaryOrigin(tx, user, ds)
	}

	protocol, fqdn, port, err := tc.ParseOrgServerFQDN(*ds.OrgServerFQDN)
	if err != nil {
		return fmt.Errorf("updating primary origin: %v", err)
	}

	name := ""
	q = `UPDATE origin SET protocol = $1, fqdn = $2, port = $3 WHERE is_primary AND deliveryservice = $4 RETURNING name`
	if err := tx.QueryRow(q, protocol, fqdn, port, *ds.ID).Scan(&name); err != nil {
		return fmt.Errorf("update primary origin for ds %s from '%s': %s", *ds.XMLID, *ds.OrgServerFQDN, err.Error())
	}

	api.CreateChangeLogRawTx(api.ApiChange, "Updated primary origin: "+name+" for delivery service: "+*ds.XMLID, user, tx)

	return nil
}

func createPrimaryOrigin(tx *sql.Tx, user *auth.CurrentUser, ds tc.DeliveryServiceNullable) error {
	if ds.OrgServerFQDN == nil {
		return nil
	}

	protocol, fqdn, port, err := tc.ParseOrgServerFQDN(*ds.OrgServerFQDN)
	if err != nil {
		return fmt.Errorf("creating primary origin: %v", err)
	}

	originID := 0
	q := `INSERT INTO origin (name, fqdn, protocol, is_primary, port, deliveryservice, tenant) VALUES ($1, $2, $3, TRUE, $4, $5, $6) RETURNING id`
	if err := tx.QueryRow(q, ds.XMLID, fqdn, protocol, port, ds.ID, ds.TenantID).Scan(&originID); err != nil {
		return fmt.Errorf("insert origin from '%s': %s", *ds.OrgServerFQDN, err.Error())
	}

	api.CreateChangeLogRawTx(api.ApiChange, "Created primary origin id: "+strconv.Itoa(originID)+" for delivery service: "+*ds.XMLID, user, tx)

	return nil
}

func getDSType(tx *sql.Tx, xmlid string) (tc.DSType, bool, error) {
	name := ""
	if err := tx.QueryRow(`SELECT name FROM type WHERE id = (select type from deliveryservice where xml_id = $1)`, xmlid).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("querying deliveryservice type name: " + err.Error())
	}
	return tc.DSTypeFromString(name), true, nil
}

func GetDeliveryServices(query string, queryValues map[string]interface{}, tx *sqlx.Tx) ([]tc.DeliveryServiceNullable, []error, tc.ApiErrorType) {
	rows, err := tx.NamedQuery(query, queryValues)
	if err != nil {
		return nil, []error{fmt.Errorf("querying: %v", err)}, tc.SystemError
	}
	defer rows.Close()

	dses := []tc.DeliveryServiceNullable{}
	dsCDNDomains := map[string]string{}

	type DSRead struct {
		tc.DeliveryServiceNullable
		CDNDomain string `db:"cdn_domain"`
	}

	for rows.Next() {
		dsRead := DSRead{}
		if err := rows.StructScan(&dsRead); err != nil {
			return nil, []error{fmt.Errorf("getting delivery services: %v", err)}, tc.SystemError
		}
		ds := dsRead.DeliveryServiceNullable
		dsCDNDomains[*ds.XMLID] = dsRead.CDNDomain
		ds.Signed = ds.SigningAlgorithm != nil && *ds.SigningAlgorithm == tc.SigningAlgorithmURLSig
		dses = append(dses, ds)
	}

	dsNames := make([]string, len(dses), len(dses))
	for i, ds := range dses {
		dsNames[i] = *ds.XMLID
	}

	matchLists, err := GetDeliveryServicesMatchLists(dsNames, tx.Tx)
	if err != nil {
		return nil, []error{errors.New("getting delivery service matchlists: " + err.Error())}, tc.SystemError
	}
	for i, ds := range dses {
		matchList, ok := matchLists[*ds.XMLID]
		if !ok {
			continue
		}
		ds.MatchList = &matchList

		ds.ExampleURLs = MakeExampleURLs(ds.Protocol, *ds.Type, *ds.RoutingName, *ds.MatchList, dsCDNDomains[*ds.XMLID])
		dses[i] = ds
	}

	return dses, nil, tc.NoError
}

func updateSSLKeys(ds *tc.DeliveryServiceNullable, hostName string, tx *sql.Tx, cfg *config.Config) error {
	if ds.XMLID == nil {
		return errors.New("delivery services has no XMLID!")
	}
	key, ok, err := riaksvc.GetDeliveryServiceSSLKeysObj(*ds.XMLID, riaksvc.DSSSLKeyVersionLatest, tx, cfg.RiakAuthOptions, cfg.RiakPort)
	if err != nil {
		return errors.New("getting SSL key: " + err.Error())
	}
	if !ok {
		return nil // no keys to update
	}
	key.DeliveryService = *ds.XMLID
	key.Hostname = hostName
	if err := riaksvc.PutDeliveryServiceSSLKeysObj(key, tx, cfg.RiakAuthOptions, cfg.RiakPort); err != nil {
		return errors.New("putting updated SSL key: " + err.Error())
	}
	return nil
}

// getHostName gets the host name used for delivery service requests. The dsProtocol may be nil, if the delivery service type doesn't have a protocol (e.g. ANY_MAP).
func getHostName(dsProtocol *int, dsType tc.DSType, dsRoutingName string, dsMatchList []tc.DeliveryServiceMatch, cdnDomain string) (string, error) {
	exampleURLs := MakeExampleURLs(dsProtocol, dsType, dsRoutingName, dsMatchList, cdnDomain)

	exampleURL := ""
	if dsProtocol != nil && *dsProtocol == 2 {
		if len(exampleURLs) < 2 {
			return "", errors.New("missing example URLs (does your delivery service have matchsets?)")
		}
		exampleURL = exampleURLs[1]
	} else {
		if len(exampleURLs) < 1 {
			return "", errors.New("missing example URLs (does your delivery service have matchsets?)")
		}
		exampleURL = exampleURLs[0]
	}

	host := strings.NewReplacer(`http://`, ``, `https://`, ``).Replace(exampleURL)
	if dsType.IsHTTP() {
		if firstDot := strings.Index(host, "."); firstDot == -1 {
			host = "*" // TODO warn? error?
		} else {
			host = "*" + host[firstDot:]
		}
	}
	return host, nil
}

func getCDNDomain(dsID int, tx *sql.Tx) (string, error) {
	q := `SELECT cdn.domain_name from cdn where cdn.id = (SELECT ds.cdn_id from deliveryservice as ds where ds.id = $1)`
	cdnDomain := ""
	if err := tx.QueryRow(q, dsID).Scan(&cdnDomain); err != nil {
		return "", fmt.Errorf("getting CDN domain for delivery service '%v': "+err.Error(), dsID)
	}
	return cdnDomain, nil
}

func getCDNNameDomainDNSSecEnabled(dsID int, tx *sql.Tx) (string, string, bool, error) {
	q := `SELECT cdn.name, cdn.domain_name, cdn.dnssec_enabled from cdn where cdn.id = (SELECT ds.cdn_id from deliveryservice as ds where ds.id = $1)`
	cdnName := ""
	cdnDomain := ""
	dnssecEnabled := false
	if err := tx.QueryRow(q, dsID).Scan(&cdnName, &cdnDomain, &dnssecEnabled); err != nil {
		return "", "", false, fmt.Errorf("getting dnssec_enabled for delivery service '%v': "+err.Error(), dsID)
	}
	return cdnName, cdnDomain, dnssecEnabled, nil
}

// makeExampleURLs creates the example URLs for a delivery service. The dsProtocol may be nil, if the delivery service type doesn't have a protocol (e.g. ANY_MAP).
func MakeExampleURLs(protocol *int, dsType tc.DSType, routingName string, matchList []tc.DeliveryServiceMatch, cdnDomain string) []string {
	examples := []string{}
	scheme := ""
	scheme2 := ""
	if protocol != nil {
		switch *protocol {
		case 0:
			scheme = "http"
		case 1:
			scheme = "https"
		case 2:
			fallthrough
		case 3:
			scheme = "http"
			scheme2 = "https"
		default:
			scheme = "http"
		}
	} else {
		scheme = "http"
	}
	dsIsDNS := dsType.IsDNS()
	regexReplacer := strings.NewReplacer(`\`, ``, `.*`, ``, `.`, ``)
	for _, match := range matchList {
		if dsIsDNS || match.Type == tc.DSMatchTypeHostRegex {
			host := regexReplacer.Replace(match.Pattern)
			if match.SetNumber == 0 {
				examples = append(examples, scheme+`://`+routingName+`.`+host+`.`+cdnDomain)
				if scheme2 != "" {
					examples = append(examples, scheme2+`://`+routingName+`.`+host+`.`+cdnDomain)
				}
				continue
			}
			examples = append(examples, scheme+`://`+match.Pattern)
			if scheme2 != "" {
				examples = append(examples, scheme2+`://`+match.Pattern)
			}
		} else if match.Type == tc.DSMatchTypePathRegex {
			examples = append(examples, match.Pattern)
		}
	}
	return examples
}

func GetDeliveryServicesMatchLists(dses []string, tx *sql.Tx) (map[string][]tc.DeliveryServiceMatch, error) {
	// TODO move somewhere generic
	q := `
SELECT ds.xml_id as ds_name, t.name as type, r.pattern, COALESCE(dsr.set_number, 0)
FROM regex as r
JOIN deliveryservice_regex as dsr ON dsr.regex = r.id
JOIN deliveryservice as ds on ds.id = dsr.deliveryservice
JOIN type as t ON r.type = t.id
WHERE ds.xml_id = ANY($1)
ORDER BY dsr.set_number
`
	rows, err := tx.Query(q, pq.Array(dses))
	if err != nil {
		return nil, errors.New("getting delivery service regexes: " + err.Error())
	}
	defer rows.Close()

	matches := map[string][]tc.DeliveryServiceMatch{}
	for rows.Next() {
		m := tc.DeliveryServiceMatch{}
		dsName := ""
		matchTypeStr := ""
		if err := rows.Scan(&dsName, &matchTypeStr, &m.Pattern, &m.SetNumber); err != nil {
			return nil, errors.New("scanning delivery service regexes: " + err.Error())
		}
		matchType := tc.DSMatchTypeFromString(matchTypeStr)
		if matchType == tc.DSMatchTypeInvalid {
			return nil, errors.New("getting delivery service regexes: got invalid delivery service match type '" + matchTypeStr + "'")
		}
		m.Type = matchType
		matches[dsName] = append(matches[dsName], m)
	}
	return matches, nil
}

type tierType int

const (
	midTier tierType = iota
	edgeTier
)

// EnsureParams ensures the given delivery service's necessary parameters exist on profiles of servers assigned to the delivery service.
// Note the edgeHeaderRewrite, midHeaderRewrite, regexRemap, and cacheURL may be nil, if the delivery service does not have those values.
func EnsureParams(tx *sql.Tx, dsID int, xmlID string, edgeHeaderRewrite *string, midHeaderRewrite *string, regexRemap *string, cacheURL *string, signingAlgorithm *string, dsType tc.DSType, maxOriginConns *int) error {
	if err := ensureHeaderRewriteParams(tx, dsID, xmlID, edgeHeaderRewrite, edgeTier, dsType, maxOriginConns); err != nil {
		return errors.New("creating edge header rewrite parameters: " + err.Error())
	}
	if err := ensureHeaderRewriteParams(tx, dsID, xmlID, midHeaderRewrite, midTier, dsType, maxOriginConns); err != nil {
		return errors.New("creating mid header rewrite parameters: " + err.Error())
	}
	if err := ensureRegexRemapParams(tx, dsID, xmlID, regexRemap); err != nil {
		return errors.New("creating mid regex remap parameters: " + err.Error())
	}
	if err := ensureCacheURLParams(tx, dsID, xmlID, cacheURL); err != nil {
		return errors.New("creating mid cacheurl parameters: " + err.Error())
	}
	if err := ensureURLSigParams(tx, dsID, xmlID, signingAlgorithm); err != nil {
		return errors.New("creating urlsig parameters: " + err.Error())
	}
	return nil
}

func ensureHeaderRewriteParams(tx *sql.Tx, dsID int, xmlID string, hdrRW *string, tier tierType, dsType tc.DSType, maxOriginConns *int) error {
	configFile := "hdr_rw_" + xmlID + ".config"
	if tier == midTier {
		configFile = "hdr_rw_mid_" + xmlID + ".config"
	}

	if tier == midTier && dsType.IsLive() && !dsType.IsNational() {
		// live local DSes don't get header rewrite rules on the mid so cleanup any location params related to mids
		return deleteLocationParam(tx, configFile)
	}

	hasMaxOriginConns := *maxOriginConns > 0 && ((tier == midTier) == dsType.UsesMidCache())
	if (hdrRW == nil || *hdrRW == "") && !hasMaxOriginConns {
		return deleteLocationParam(tx, configFile)
	}
	locationParamID, err := ensureLocation(tx, configFile)
	if err != nil {
		return err
	}
	if tier != midTier {
		return createDSLocationProfileParams(tx, locationParamID, dsID)
	}
	profileParameterQuery := `
INSERT INTO profile_parameter (profile, parameter)
SELECT DISTINCT(profile), $1::bigint FROM server
WHERE server.type IN (SELECT id from type where type.name like 'MID%' and type.use_in_table = 'server')
AND server.cdn_id = (select cdn_id from deliveryservice where id = $2)
ON CONFLICT DO NOTHING
`
	if _, err := tx.Exec(profileParameterQuery, locationParamID, dsID); err != nil {
		return fmt.Errorf("parameter query to insert profile_parameters query '"+profileParameterQuery+"' location parameter ID '%v' delivery service ID '%v': %v", locationParamID, dsID, err)
	}
	return nil
}

func ensureURLSigParams(tx *sql.Tx, dsID int, xmlID string, signingAlgorithm *string) error {
	configFile := "url_sig_" + xmlID + ".config"
	if signingAlgorithm == nil || *signingAlgorithm != tc.SigningAlgorithmURLSig {
		return deleteLocationParam(tx, configFile)
	}
	locationParamID, err := ensureLocation(tx, configFile)
	if err != nil {
		return err
	}
	return createDSLocationProfileParams(tx, locationParamID, dsID)
}

func ensureRegexRemapParams(tx *sql.Tx, dsID int, xmlID string, regexRemap *string) error {
	configFile := "regex_remap_" + xmlID + ".config"
	if regexRemap == nil || *regexRemap == "" {
		return deleteLocationParam(tx, configFile)
	}
	locationParamID, err := ensureLocation(tx, configFile)
	if err != nil {
		return err
	}
	return createDSLocationProfileParams(tx, locationParamID, dsID)
}

func ensureCacheURLParams(tx *sql.Tx, dsID int, xmlID string, cacheURL *string) error {
	configFile := "cacheurl_" + xmlID + ".config"
	if cacheURL == nil || *cacheURL == "" {
		return deleteLocationParam(tx, configFile)
	}
	locationParamID, err := ensureLocation(tx, configFile)
	if err != nil {
		return err
	}
	return createDSLocationProfileParams(tx, locationParamID, dsID)
}

// createDSLocationProfileParams adds the given parameter to all profiles assigned to servers which are assigned to the given delivery service.
func createDSLocationProfileParams(tx *sql.Tx, locationParamID int, deliveryServiceID int) error {
	profileParameterQuery := `
INSERT INTO profile_parameter (profile, parameter)
SELECT DISTINCT(profile), $1::bigint FROM server
WHERE server.id IN (SELECT server from deliveryservice_server where deliveryservice = $2)
ON CONFLICT DO NOTHING
`
	if _, err := tx.Exec(profileParameterQuery, locationParamID, deliveryServiceID); err != nil {
		return errors.New("inserting profile_parameters: " + err.Error())
	}
	return nil
}

// ensureLocation ensures a location parameter exists for the given config file. If not, it creates one, with the same value as the 'remap.config' file parameter. Returns the ID of the location parameter.
func ensureLocation(tx *sql.Tx, configFile string) (int, error) {
	atsConfigLocation := ""
	if err := tx.QueryRow(`SELECT value FROM parameter WHERE name = 'location' AND config_file = 'remap.config'`).Scan(&atsConfigLocation); err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("executing parameter query for ATS config location: parameter missing (do you have a name=location config_file=remap.config parameter?")
		}
		return 0, errors.New("executing parameter query for ATS config location: " + err.Error())
	}
	atsConfigLocation = strings.TrimRight(atsConfigLocation, `/`)

	locationParamID := 0
	existingLocationErr := tx.QueryRow(`SELECT id FROM parameter WHERE name = 'location' AND config_file = $1`, configFile).Scan(&locationParamID)
	if existingLocationErr != nil && existingLocationErr != sql.ErrNoRows {
		return 0, errors.New("executing parameter query for existing location: " + existingLocationErr.Error())
	}

	if existingLocationErr == sql.ErrNoRows {
		resultRows, err := tx.Query(`INSERT INTO parameter (config_file, name, value) VALUES ($1, 'location', $2) RETURNING id`, configFile, atsConfigLocation)
		if err != nil {
			return 0, errors.New("executing parameter query to insert location: " + err.Error())
		}
		defer resultRows.Close()
		if !resultRows.Next() {
			return 0, errors.New("parameter query to insert location didn't return id")
		}
		if err := resultRows.Scan(&locationParamID); err != nil {
			return 0, errors.New("parameter query to insert location returned id scan: " + err.Error())
		}
		if resultRows.Next() {
			return 0, errors.New("parameter query to insert location returned too many rows (>1)")
		}
	}
	return locationParamID, nil
}

func deleteLocationParam(tx *sql.Tx, configFile string) error {
	id := 0
	err := tx.QueryRow(`DELETE FROM parameter WHERE name = 'location' AND config_file = $1 RETURNING id`, configFile).Scan(&id)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		log.Errorln("deleting name=location config_file=" + configFile + " parameter: " + err.Error())
		return errors.New("executing parameter delete: " + err.Error())
	}
	if _, err := tx.Exec(`DELETE FROM profile_parameter WHERE parameter = $1`, id); err != nil {
		log.Errorf("deleting parameter name=location config_file=%v id=%v profile_parameter: %v", configFile, id, err)
		return errors.New("executing parameter profile_parameter delete: " + err.Error())
	}
	return nil
}

// getTenantID returns the tenant Id of the given delivery service. Note it may return a nil id and nil error, if the tenant ID in the database is nil.
func getTenantID(tx *sql.Tx, ds *tc.DeliveryServiceNullable) (*int, error) {
	if ds.ID == nil && ds.XMLID == nil {
		return nil, errors.New("delivery service has no ID or XMLID")
	}
	if ds.ID != nil {
		existingID, _, err := getDSTenantIDByID(tx, *ds.ID) // ignore exists return - if the DS is new, we only need to check the user input tenant
		return existingID, err
	}
	existingID, _, err := getDSTenantIDByName(tx, *ds.XMLID) // ignore exists return - if the DS is new, we only need to check the user input tenant
	return existingID, err
}

func isTenantAuthorized(inf *api.APIInfo, ds *tc.DeliveryServiceNullable) (bool, error) {
	tx := inf.Tx.Tx
	user := inf.User

	existingID, err := getTenantID(inf.Tx.Tx, ds)
	if err != nil {
		return false, errors.New("getting tenant ID: " + err.Error())
	}
	if ds.TenantID == nil {
		ds.TenantID = existingID
	}
	if existingID != nil && existingID != ds.TenantID {
		userAuthorizedForExistingDSTenant, err := tenant.IsResourceAuthorizedToUserTx(*existingID, user, tx)
		if err != nil {
			return false, errors.New("checking authorization for existing DS ID: " + err.Error())
		}
		if !userAuthorizedForExistingDSTenant {
			return false, nil
		}
	}
	if ds.TenantID != nil {
		userAuthorizedForNewDSTenant, err := tenant.IsResourceAuthorizedToUserTx(*ds.TenantID, user, tx)
		if err != nil {
			return false, errors.New("checking authorization for new DS ID: " + err.Error())
		}
		if !userAuthorizedForNewDSTenant {
			return false, nil
		}
	}
	return true, nil
}

// getDSTenantIDByID returns the tenant ID, whether the delivery service exists, and any error.
func getDSTenantIDByID(tx *sql.Tx, id int) (*int, bool, error) {
	tenantID := (*int)(nil)
	if err := tx.QueryRow(`SELECT tenant_id FROM deliveryservice where id = $1`, id).Scan(&tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("querying tenant ID for delivery service ID '%v': %v", id, err)
	}
	return tenantID, true, nil
}

// GetDSTenantIDByIDTx returns the tenant ID, whether the delivery service exists, and any error.
func GetDSTenantIDByIDTx(tx *sql.Tx, id int) (*int, bool, error) {
	tenantID := (*int)(nil)
	if err := tx.QueryRow(`SELECT tenant_id FROM deliveryservice where id = $1`, id).Scan(&tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("querying tenant ID for delivery service ID '%v': %v", id, err)
	}
	return tenantID, true, nil
}

// getDSTenantIDByName returns the tenant ID, whether the delivery service exists, and any error.
func getDSTenantIDByName(tx *sql.Tx, name string) (*int, bool, error) {
	tenantID := (*int)(nil)
	if err := tx.QueryRow(`SELECT tenant_id FROM deliveryservice where xml_id = $1`, name).Scan(&tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("querying tenant ID for delivery service name '%v': %v", name, err)
	}
	return tenantID, true, nil
}

// GetDSTenantIDByNameTx returns the tenant ID, whether the delivery service exists, and any error.
func GetDSTenantIDByNameTx(tx *sql.Tx, ds tc.DeliveryServiceName) (*int, bool, error) {
	tenantID := (*int)(nil)
	if err := tx.QueryRow(`SELECT tenant_id FROM deliveryservice where xml_id = $1`, ds).Scan(&tenantID); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("querying tenant ID for delivery service name '%v': %v", ds, err)
	}
	return tenantID, true, nil
}

// GetDeliveryServiceType returns the type of the deliveryservice.
func GetDeliveryServiceType(dsID int, tx *sql.Tx) (tc.DSType, error) {
	var dsType tc.DSType
	if err := tx.QueryRow(`SELECT t.name FROM deliveryservice as ds JOIN type t ON ds.type = t.id WHERE ds.id=$1`, dsID).Scan(&dsType); err != nil {
		if err == sql.ErrNoRows {
			return tc.DSTypeInvalid, errors.New("a deliveryservice with id '" + strconv.Itoa(dsID) + "' was not found")
		}
		return tc.DSTypeInvalid, errors.New("querying type from delivery service: " + err.Error())
	}
	return dsType, nil
}

// GetXMLID loads the DeliveryService's xml_id from the database, from the ID. Returns whether the delivery service was found, and any error.
func GetXMLID(tx *sql.Tx, id int) (string, bool, error) {
	xmlID := ""
	if err := tx.QueryRow(`SELECT xml_id FROM deliveryservice where id = $1`, id).Scan(&xmlID); err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("querying xml_id for delivery service ID '%v': %v", id, err)
	}
	return xmlID, true, nil
}

// DBJoinFields is a map of fields used in the DeliveryService object, which come from other tables.
func DBJoinFields() map[string]string {
	return map[string]string{
		`org_server_fqdn`: `
		  (SELECT o.protocol::::text || ':://' || o.fqdn || rtrim(concat('::', o.port::::text), '::')
		  FROM origin o
		  WHERE o.deliveryservice = ds.id
		  AND o.is_primary) AS org_server_fqdn`,
		`profile_name`:        `profile.name AS profile_name`,
		`profile_description`: `profile.description AS profile_description`,
		`tenant_name`:         `tenant.name AS tenant_name`,
		`type_name`:           `type.name AS type_name`,
		`cdn_domain`:          `cdn.domain_name AS cdn_domain`,
		`cdn_name`:            `cdn.name as cdn_name`,
	}
}

// DBOmitModifyFields is a map of fields which should be omitted on insert or update.
// These are generally fields from join tables, the id, and last_updated.
func DBOmitModifyFields() map[string]string {
	omitFields := DBJoinFields()
	omitFields["id"] = ""
	omitFields["last_updated"] = ""
	return omitFields
}

// DBTransformFields is a map of fields used in the DeliveryService object,
// which are on the delivery service table, but are transformed in some way from the raw data.
func DBTransformFields(tableAlias string) map[string]string {
	if tableAlias != "" {
		tableAlias = tableAlias + "."
	}
	return map[string]string{
		`deep_caching_type`: `CAST(` + tableAlias + `deep_caching_type AS text) as deep_caching_type`,
		`miss_lat`:          `COALESCE(` + tableAlias + `miss_lat, 0.0) AS miss_lat`,
		`miss_long`:         `COALESCE(` + tableAlias + `miss_long, 0.0) AS miss_long`,
	}
}

func SelectQuery(ds tc.IDeliveryServiceNullable) string {
	allFields := ds.DBFields()
	tableAlias := `ds`
	transformFields := DBTransformFields(tableAlias)
	joinFields := DBJoinFields()
	fields := []string{}
	for _, field := range allFields {
		if replacement, ok := joinFields[field]; ok {
			field = replacement
		} else if replacement, ok := transformFields[field]; ok {
			field = replacement
		} else {
			field = "ds." + field
		}
		if field == "" {
			continue
		}
		fields = append(fields, field)
	}
	return `
SELECT
  ` + strings.Join(fields, ",") + `
FROM
  ` + ds.DBTable() + ` AS ds
  JOIN type ON ds.type = type.id
  JOIN cdn ON ds.cdn_id = cdn.id
  LEFT JOIN profile ON ds.profile = profile.id
  LEFT JOIN tenant ON ds.tenant_id = tenant.id
`
}

func InsertQuery(ds *tc.DeliveryServiceNullable) string {
	fields := dbhelpers.RemoveDBFields(ds.DBFields(), DBOmitModifyFields())
	return `
INSERT INTO ` + ds.DBTable() + `
(` + strings.Join(fields, ",") + `)
VALUES (` + strings.Join(util.MapStr(fields, func(s string) string { return ":" + s }), ",") + `)
RETURNING id, last_updated
`
}

func UpdateQuery(ds tc.DBer) string {
	fields := dbhelpers.RemoveDBFields(ds.DBFields(), DBOmitModifyFields())
	return `
UPDATE ` + ds.DBTable() + `
SET ` + strings.Join(util.MapStr(fields, func(s string) string { return s + "=:" + s }), ",") + `
WHERE id=:id
RETURNING last_updated
`
}
