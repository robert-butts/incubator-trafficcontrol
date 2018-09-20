/*

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package toclient

import (
	"net"
	"net/url"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/client"
)

// New returns a new Traffic Ops Client.
func New(toURL string, toUser string, toPasswd string, insecure bool, userAgent string, useCache bool, requestTimeout time.Duration) (Client, net.Addr, error) {
	return client.LoginWithAgent(toURL, toUser, toPasswd, insecure, userAgent, useCache, requestTimeout)
}

// Client is a Traffic Ops client.
type Client interface {
	URI() string
	User() string

	GetAbout() (map[string]string, client.ReqInf, error)
	CreateASN(entity tc.ASN) (tc.Alerts, client.ReqInf, error)
	UpdateASNByID(id int, entity tc.ASN) (tc.Alerts, client.ReqInf, error)
	GetASNs() ([]tc.ASN, client.ReqInf, error)
	GetASNByID(id int) ([]tc.ASN, client.ReqInf, error)
	GetASNByASN(asn int) ([]tc.ASN, client.ReqInf, error)
	DeleteASNByASN(asn int) (tc.Alerts, client.ReqInf, error)
	DeleteDeliveryServiceServer(dsID int, serverID int) (tc.Alerts, client.ReqInf, error)
	GetDeliveryServiceServers() (tc.DeliveryServiceServerResponse, client.ReqInf, error)

	CreateCacheGroupNullable(cachegroup tc.CacheGroupNullable) (*tc.CacheGroupDetailResponse, client.ReqInf, error)
	UpdateCacheGroupNullableByID(id int, cachegroup tc.CacheGroupNullable) (*tc.CacheGroupDetailResponse, client.ReqInf, error)
	GetCacheGroupsNullable() ([]tc.CacheGroupNullable, client.ReqInf, error)
	GetCacheGroupNullableByID(id int) ([]tc.CacheGroupNullable, client.ReqInf, error)
	GetCacheGroupNullableByName(name string) ([]tc.CacheGroupNullable, client.ReqInf, error)
	DeleteCacheGroupByID(id int) (tc.Alerts, client.ReqInf, error)

	CreateCDN(cdn tc.CDN) (tc.Alerts, client.ReqInf, error)
	UpdateCDNByID(id int, cdn tc.CDN) (tc.Alerts, client.ReqInf, error)
	GetCDNs() ([]tc.CDN, client.ReqInf, error)
	GetCDNByID(id int) ([]tc.CDN, client.ReqInf, error)
	GetCDNByName(name string) ([]tc.CDN, client.ReqInf, error)
	DeleteCDNByID(id int) (tc.Alerts, client.ReqInf, error)
	GetCDNSSLKeys(name string) ([]tc.CDNSSLKeys, client.ReqInf, error)

	GetDomains() ([]tc.Domain, client.ReqInf, error)

	CreateCDNFederationByName(f tc.CDNFederation, CDNName string) (*tc.CreateCDNFederationResponse, client.ReqInf, error)
	GetCDNFederationsByName(CDNName string) (*tc.CDNFederationResponse, client.ReqInf, error)
	GetCDNFederationsByID(CDNName string, ID int) (*tc.CDNFederationResponse, client.ReqInf, error)
	UpdateCDNFederationsByID(f tc.CDNFederation, CDNName string, ID int) (*tc.UpdateCDNFederationResponse, client.ReqInf, error)
	DeleteCDNFederationByID(CDNName string, ID int) (*tc.DeleteCDNFederationResponse, client.ReqInf, error)

	CreateCoordinate(coordinate tc.Coordinate) (tc.Alerts, client.ReqInf, error)
	UpdateCoordinateByID(id int, coordinate tc.Coordinate) (tc.Alerts, client.ReqInf, error)
	GetCoordinates() ([]tc.Coordinate, client.ReqInf, error)
	GetCoordinateByID(id int) ([]tc.Coordinate, client.ReqInf, error)
	GetCoordinateByName(name string) ([]tc.Coordinate, client.ReqInf, error)
	DeleteCoordinateByID(id int) (tc.Alerts, client.ReqInf, error)
	SetCachegroupDeliveryServices(cgID int, dsIDs []int64) (tc.CacheGroupPostDSRespResponse, client.ReqInf, error)

	GetCRConfig(cdn string) ([]byte, client.ReqInf, error)

	GetDeliveryServices() ([]tc.DeliveryService, client.ReqInf, error)
	GetDeliveryServicesByServer(id int) ([]tc.DeliveryService, client.ReqInf, error)
	GetDeliveryServiceByXMLID(XMLID string) ([]tc.DeliveryService, client.ReqInf, error)
	GetDeliveryService(id string) (*tc.DeliveryService, client.ReqInf, error)

	CreateDeliveryService(ds *tc.DeliveryService) (*tc.CreateDeliveryServiceResponse, error)
	UpdateDeliveryService(id string, ds *tc.DeliveryService) (*tc.UpdateDeliveryServiceResponse, error)
	DeleteDeliveryService(id string) (*tc.DeleteDeliveryServiceResponse, error)

	GetDeliveryServiceState(id string) (*tc.DeliveryServiceState, client.ReqInf, error)
	GetDeliveryServiceHealth(id string) (*tc.DeliveryServiceHealth, client.ReqInf, error)
	GetDeliveryServiceCapacity(id string) (*tc.DeliveryServiceCapacity, client.ReqInf, error)
	GetDeliveryServiceRouting(id string) (*tc.DeliveryServiceRouting, client.ReqInf, error)
	GetDeliveryServiceServer(page, limit string) ([]tc.DeliveryServiceServer, client.ReqInf, error)
	GetDeliveryServiceRegexes() ([]tc.DeliveryServiceRegexes, client.ReqInf, error)
	GetDeliveryServiceSSLKeysByID(id string) (*tc.DeliveryServiceSSLKeys, client.ReqInf, error)
	GetDeliveryServiceSSLKeysByHostname(hostname string) (*tc.DeliveryServiceSSLKeys, client.ReqInf, error)
	GetDeliveryServiceMatches() ([]tc.DeliveryServicePatterns, client.ReqInf, error)
	GetDeliveryServicesEligible(dsID int) ([]tc.DSServer, client.ReqInf, error)

	CreateDeliveryServiceRequestComment(comment tc.DeliveryServiceRequestComment) (tc.Alerts, client.ReqInf, error)
	UpdateDeliveryServiceRequestCommentByID(id int, comment tc.DeliveryServiceRequestComment) (tc.Alerts, client.ReqInf, error)
	GetDeliveryServiceRequestComments() ([]tc.DeliveryServiceRequestComment, client.ReqInf, error)
	GetDeliveryServiceRequestCommentByID(id int) ([]tc.DeliveryServiceRequestComment, client.ReqInf, error)
	DeleteDeliveryServiceRequestCommentByID(id int) (tc.Alerts, client.ReqInf, error)

	CreateDeliveryServiceRequest(dsr tc.DeliveryServiceRequest) (tc.Alerts, client.ReqInf, error)
	GetDeliveryServiceRequests() ([]tc.DeliveryServiceRequest, client.ReqInf, error)
	GetDeliveryServiceRequestByXMLID(XMLID string) ([]tc.DeliveryServiceRequest, client.ReqInf, error)
	GetDeliveryServiceRequestByID(id int) ([]tc.DeliveryServiceRequest, client.ReqInf, error)
	UpdateDeliveryServiceRequestByID(id int, dsr tc.DeliveryServiceRequest) (tc.Alerts, client.ReqInf, error)
	DeleteDeliveryServiceRequestByID(id int) (tc.Alerts, client.ReqInf, error)

	CreateDeliveryServiceServers(dsID int, serverIDs []int, replace bool) (*tc.DSServerIDs, error)

	CreateDivision(division tc.Division) (tc.Alerts, client.ReqInf, error)
	UpdateDivisionByID(id int, division tc.Division) (tc.Alerts, client.ReqInf, error)
	GetDivisions() ([]tc.Division, client.ReqInf, error)
	GetDivisionByID(id int) ([]tc.Division, client.ReqInf, error)
	GetDivisionByName(name string) ([]tc.Division, client.ReqInf, error)
	DeleteDivisionByID(id int) (tc.Alerts, client.ReqInf, error)
	DeleteDivisionByName(name string) (tc.Alerts, client.ReqInf, error)

	GetUserDeliveryServices(userID int) (*tc.UserDeliveryServicesNullableResponse, client.ReqInf, error)

	SetDeliveryServiceUser(userID int, dses []int, replace bool) (*tc.UserDeliveryServicePostResponse, error)
	DeleteDeliveryServiceUser(userID int, dsID int) (*tc.UserDeliveryServiceDeleteResponse, error)

	GetHardware(limit int) ([]tc.Hardware, client.ReqInf, error)

	CreateOrigin(origin tc.Origin) (*tc.OriginDetailResponse, client.ReqInf, error)
	UpdateOriginByID(id int, origin tc.Origin) (*tc.OriginDetailResponse, client.ReqInf, error)
	GetOriginsByQueryParams(queryParams string) ([]tc.Origin, client.ReqInf, error)
	GetOrigins() ([]tc.Origin, client.ReqInf, error)
	GetOriginByID(id int) ([]tc.Origin, client.ReqInf, error)
	GetOriginByName(name string) ([]tc.Origin, client.ReqInf, error)
	GetOriginsByDeliveryServiceID(id int) ([]tc.Origin, client.ReqInf, error)
	DeleteOriginByID(id int) (tc.Alerts, client.ReqInf, error)

	CreateParameter(pl tc.Parameter) (tc.Alerts, client.ReqInf, error)
	UpdateParameterByID(id int, pl tc.Parameter) (tc.Alerts, client.ReqInf, error)
	GetParameters() ([]tc.Parameter, client.ReqInf, error)
	Parameters(profileName string) ([]tc.Parameter, error)
	GetParametersByProfileName(profileName string) ([]tc.Parameter, client.ReqInf, error)
	GetParameterByID(id int) ([]tc.Parameter, client.ReqInf, error)
	GetParameterByName(name string) ([]tc.Parameter, client.ReqInf, error)
	GetParameterByConfigFile(configFile string) ([]tc.Parameter, client.ReqInf, error)
	GetParameterByNameAndConfigFile(name string, configFile string) ([]tc.Parameter, client.ReqInf, error)
	DeleteParameterByID(id int) (tc.Alerts, client.ReqInf, error)

	CreatePhysLocation(pl tc.PhysLocation) (tc.Alerts, client.ReqInf, error)
	UpdatePhysLocationByID(id int, pl tc.PhysLocation) (tc.Alerts, client.ReqInf, error)
	GetPhysLocations() ([]tc.PhysLocation, client.ReqInf, error)
	GetPhysLocationByID(id int) ([]tc.PhysLocation, client.ReqInf, error)
	GetPhysLocationByName(name string) ([]tc.PhysLocation, client.ReqInf, error)
	DeletePhysLocationByID(id int) (tc.Alerts, client.ReqInf, error)

	Ping() (map[string]string, client.ReqInf, error)

	CreateProfile(pl tc.Profile) (tc.Alerts, client.ReqInf, error)
	UpdateProfileByID(id int, pl tc.Profile) (tc.Alerts, client.ReqInf, error)
	GetProfiles() ([]tc.Profile, client.ReqInf, error)
	GetProfileByID(id int) ([]tc.Profile, client.ReqInf, error)
	GetProfileByName(name string) ([]tc.Profile, client.ReqInf, error)
	GetProfileByParameter(param string) ([]tc.Profile, client.ReqInf, error)
	GetProfileByCDNID(cdnID int) ([]tc.Profile, client.ReqInf, error)
	DeleteProfileByID(id int) (tc.Alerts, client.ReqInf, error)
	CreateProfileParameter(pp tc.ProfileParameter) (tc.Alerts, client.ReqInf, error)
	GetProfileParameters() ([]tc.ProfileParameter, client.ReqInf, error)
	GetProfileParameterByQueryParams(queryParams string) ([]tc.ProfileParameter, client.ReqInf, error)
	DeleteParameterByProfileParameter(profile int, parameter int) (tc.Alerts, client.ReqInf, error)
	CreateRegion(region tc.Region) (tc.Alerts, client.ReqInf, error)
	UpdateRegionByID(id int, region tc.Region) (tc.Alerts, client.ReqInf, error)
	GetRegions() ([]tc.Region, client.ReqInf, error)
	GetRegionByID(id int) ([]tc.Region, client.ReqInf, error)
	GetRegionByName(name string) ([]tc.Region, client.ReqInf, error)
	DeleteRegionByID(id int) (tc.Alerts, client.ReqInf, error)
	GetRegionByNamePath(name string) ([]tc.RegionName, client.ReqInf, error)

	CreateRole(region tc.Role) (tc.Alerts, client.ReqInf, int, error)
	UpdateRoleByID(id int, region tc.Role) (tc.Alerts, client.ReqInf, int, error)
	GetRoles() ([]tc.Role, client.ReqInf, int, error)
	GetRoleByID(id int) ([]tc.Role, client.ReqInf, int, error)
	GetRoleByName(name string) ([]tc.Role, client.ReqInf, int, error)
	DeleteRoleByID(id int) (tc.Alerts, client.ReqInf, int, error)

	CreateServer(server tc.Server) (tc.Alerts, client.ReqInf, error)
	UpdateServerByID(id int, server tc.Server) (tc.Alerts, client.ReqInf, error)
	GetServers() ([]tc.Server, client.ReqInf, error)
	GetServerByID(id int) ([]tc.Server, client.ReqInf, error)
	GetServerByHostName(hostName string) ([]tc.Server, client.ReqInf, error)
	DeleteServerByID(id int) (tc.Alerts, client.ReqInf, error)
	GetServersByType(qparams url.Values) ([]tc.Server, client.ReqInf, error)
	GetServerFQDN(n string) (string, client.ReqInf, error)
	GetServersShortNameSearch(shortname string) ([]string, client.ReqInf, error)

	CreateStaticDNSEntry(cdn tc.StaticDNSEntry) (tc.Alerts, client.ReqInf, error)
	UpdateStaticDNSEntryByID(id int, cdn tc.StaticDNSEntry) (tc.Alerts, client.ReqInf, int, error)
	GetStaticDNSEntries() ([]tc.StaticDNSEntry, client.ReqInf, error)
	GetStaticDNSEntryByID(id int) ([]tc.StaticDNSEntry, client.ReqInf, error)
	GetStaticDNSEntriesByHost(host string) ([]tc.StaticDNSEntry, client.ReqInf, error)
	DeleteStaticDNSEntryByID(id int) (tc.Alerts, client.ReqInf, error)

	GetSummaryStats(cdn string, deliveryService string, statName string) ([]tc.StatsSummary, client.ReqInf, error)
	GetSummaryStatsLastUpdated(statName string) (string, client.ReqInf, error)
	DoAddSummaryStats(statsSummary tc.StatsSummary) (client.ReqInf, error)

	CreateStatus(status tc.Status) (tc.Alerts, client.ReqInf, error)
	UpdateStatusByID(id int, status tc.Status) (tc.Alerts, client.ReqInf, error)
	GetStatuses() ([]tc.Status, client.ReqInf, error)
	GetStatusByID(id int) ([]tc.Status, client.ReqInf, error)
	GetStatusByName(name string) ([]tc.Status, client.ReqInf, error)
	DeleteStatusByID(id int) (tc.Alerts, client.ReqInf, error)

	CreateSteeringTarget(st tc.SteeringTargetNullable) (tc.Alerts, client.ReqInf, error)
	UpdateSteeringTarget(st tc.SteeringTargetNullable) (tc.Alerts, client.ReqInf, error)
	GetSteeringTargets(dsID int) ([]tc.SteeringTargetNullable, client.ReqInf, error)
	DeleteSteeringTarget(dsID int, targetID int) (tc.Alerts, client.ReqInf, error)

	Tenants() ([]tc.Tenant, client.ReqInf, error)
	Tenant(id string) (*tc.Tenant, client.ReqInf, error)
	TenantByName(name string) (*tc.Tenant, client.ReqInf, error)

	CreateTenant(t *tc.Tenant) (*tc.TenantResponse, error)
	UpdateTenant(id string, t *tc.Tenant) (*tc.TenantResponse, error)
	DeleteTenant(id string) (*tc.DeleteTenantResponse, error)

	GetTrafficMonitorConfigMap(cdn string) (*tc.TrafficMonitorConfigMap, client.ReqInf, error)
	GetTrafficMonitorConfig(cdn string) (*tc.TrafficMonitorConfig, client.ReqInf, error)

	CreateType(typ tc.Type) (tc.Alerts, client.ReqInf, error)
	UpdateTypeByID(id int, typ tc.Type) (tc.Alerts, client.ReqInf, error)
	GetTypes(useInTable ...string) ([]tc.Type, client.ReqInf, error)
	GetTypeByID(id int) ([]tc.Type, client.ReqInf, error)
	GetTypeByName(name string) ([]tc.Type, client.ReqInf, error)
	DeleteTypeByID(id int) (tc.Alerts, client.ReqInf, error)

	GetUpdate(serverName string) (client.Update, client.ReqInf, error)
	SetUpdate(serverName string, updatePending int, revalPending int) (client.ReqInf, error)

	GetUsers() ([]tc.User, client.ReqInf, error)
	GetUserByID(id int) ([]tc.User, client.ReqInf, error)
	GetUserByUsername(username string) ([]tc.User, client.ReqInf, error)
	GetUserCurrent() (*tc.UserCurrent, client.ReqInf, error)
	CreateUser(user *tc.User) (*tc.CreateUserResponse, client.ReqInf, error)
	UpdateUserByID(id int, u *tc.User) (*tc.UpdateUserResponse, client.ReqInf, error)
	DeleteUserByID(id int) (tc.Alerts, client.ReqInf, error)
}
