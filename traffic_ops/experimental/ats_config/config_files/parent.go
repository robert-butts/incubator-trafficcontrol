package config_files

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	towrap "github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/trafficopswrapper"
	to "github.com/apache/incubator-trafficcontrol/traffic_ops/client"
)

func createParentDotConfig(toClient towrap.ITrafficOpsSession, filename string, trafficOpsHost string, trafficServerHost string, params []to.Parameter) (string, error) {
	serverTypeStr, err := getServerTypeStr(toClient, trafficServerHost)
	if err != nil {
		return "", fmt.Errorf("error getting server '%v' type: %v", trafficServerHost, err)
	}

	// 	Origin Shield or Multi Site Origin
	// $self->app->log->debug( "id = $id and server_type = $server_type,  hostname = " . $server->{host_name} );
	if isMid := strings.HasPrefix(serverTypeStr, "MID"); isMid {
		return createParentDotConfigMid(toClient, filename, trafficOpsHost, trafficServerHost, params)
	} else {
		return createParentDotConfigEdge(toClient, filename, trafficOpsHost, trafficServerHost, params)
	}

	// if _, ok := paramMap["storage.config"]; !ok {
	// 	return "", fmt.Errorf("No storage config parameters")
	// }

	// storageConfigParams := paramMap["storage.config"]

	// volumePrefixes := []string{"", "RAM_", "SSD_"}

	// numVolumes := 0
	// for _, prefix := range volumePrefixes {
	// 	if _, ok := storageConfigParams[prefix+"Drive_Prefix"]; ok {
	// 		numVolumes++
	// 	}
	// }

	// volumeText := func(volumeNum int, numVolumes int) string {
	// 	return fmt.Sprintf("volume=%d scheme=http size=%d%%\n", volumeNum, 100/numVolumes)
	// }

	// nextVolumeNum := 1
	// for _, prefix := range volumePrefixes {
	// 	if _, hasDrivePrefix := storageConfigParams[prefix+"Drive_Prefix"]; hasDrivePrefix {
	// 		s += volumeText(nextVolumeNum, numVolumes)
	// 		nextVolumeNum++
	// 	}
	// }
	// return s, nil
}

// getUriPort returns the port for the given URI. If the URI does not include an explicit port, a best-guess is returned, following the heuristic:
// If the scheme is included, the scheme's default port is returned (80 for http, 443 for https)
// If the scheme is not included, http is assumed and 80 is returned.
// TODO test
// func getUriPort(uri string) int {
// 	defaultPort := 80
// 	if strings.HasPrefix(uri, "http://") {
// 		uri := uri[len("http://"):]
// 	} else if strings.HasPrefix(uri, "https://") {
// 		uri := uri[len("https://"):]
// 		defaultPort = 443
// 	}

// 	portI := strings.Index(uri, ":")
// 	if portI == -1 || portI == len(uri)-1 {
// 		return defaultPort
// 	}

// 	uriPortStart := uri[portI+1:]
// 	uriPortEnd := strings.Index(uriPortStart, "/")
// 	if uriPortEnd != -1 {
// 		uriPortStart = uriPortStart[:uriPortEnd-1]
// 	}
// 	uriPort, err := strconv.Atoi(uriPortStart)
// 	if err != nil {
// 		// TODO log
// 		return defaultPort
// 	}
// 	return uriPort
// }

func getParam(paramMap map[string]map[string]string, file, param string) (string, bool) {
	fileMap, ok := paramMap[file]
	if !ok {
		return "", false
	}
	val, ok := fileMap[param]
	return val, ok
}

// TODO test
func uriHost(u *url.URL) string {
	i := strings.Index(u.Host, ":")
	if i == -1 {
		return u.Host
	}
	if i == 0 {
		return ""
	}
	return u.Host[:i-1]
}

// TODO test
func uriPort(u *url.URL) string {
	i := strings.Index(u.Host, ":")
	if i != -1 && i < len(u.Host)-1 {
		return u.Host[i:]
	}

	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}

func createParentDotConfigMid(toClient towrap.ITrafficOpsSession, filename string, trafficOpsHost string, trafficServerHost string, params []to.Parameter) (string, error) {
	deliveryServices, err := toClient.DeliveryServices()
	if err != nil {
		return "", fmt.Errorf("error getting delivery services: %v", err)
	}

	paramMap := createParamsMap(params)

	uniqueOrigin := map[string]struct{}{} // TODO rename

	pinfo, err := parentData(toClient, trafficServerHost, paramMap)
	if err != nil {
		return "", fmt.Errorf("getting parent data: %v", err)
	}

	s := "# DO NOT EDIT - Generated for " + trafficServerHost + " by Traffic Ops (" + trafficOpsHost + ") on " + time.Now().String() + "\n"

	for _, ds := range deliveryServices {
		os := ds.OriginShield
		parentQstring := "ignore"
		multiSiteOrigin := ds.MultiSiteOrigin
		multiSiteOriginAlgorithm := ds.MultiSiteOriginAlgorithm
		if _, ok := uniqueOrigin[ds.OrgServerFQDN]; ok {
			continue
		}
		uniqueOrigin[ds.OrgServerFQDN] = struct{}{}

		orgUri, err := url.Parse(ds.OrgServerFQDN)
		if err != nil {
			return "", fmt.Errorf("parsing ds %v OrgServerFQDN '%s': %v", ds.XMLID, ds.OrgServerFQDN, err)
		}

		if os != "" {
			pSelectAlg, pSelectAlgExists := getParam(paramMap, "parent.config", "algorithm")
			algorithm := ""
			if pSelectAlgExists {
				algorithm = fmt.Sprintf("round_robin=%s", pSelectAlg)
			}
			s += fmt.Sprintf(`dest_domain=%s port=%d parent=%s %s go_direct=true\n
`, uriHost(orgUri), uriPort(orgUri), os, algorithm)
		} else if multiSiteOrigin {
			s += fmt.Sprintf(`dest_domain=%s  port=%d `, uriHost(orgUri), uriPort(orgUri))

			if _, ok := pinfo[uriHost(orgUri)]; !ok {
				return "", fmt.Errorf("origin URI %v (%v) not found in parent data", orgUri, uriHost(orgUri))
			}
			rankedParents := sortParentsByRank(pinfo[uriHost(orgUri)])

			parentInfo := []string{}
			secondaryParentInfo := []string{}
			nullParentInfo := []string{}
			for _, parent := range rankedParents {
				if parent.PrimaryParent {
					parentInfo = append(parentInfo, formatParentInfo(parent))
				} else if parent.SecondaryParent {
					secondaryParentInfo = append(secondaryParentInfo, formatParentInfo(parent))
				} else {
					nullParentInfo = append(nullParentInfo, formatParentInfo(parent))
				}
			}

			parentInfo = uniqStrs(parentInfo)
			secondaryParentInfo = uniqStrs(secondaryParentInfo)
			nullParentInfo = uniqStrs(nullParentInfo)

			parents := fmt.Sprintf(`parent="%s%s%s"`, strings.Join(parentInfo, ""), strings.Join(secondaryParentInfo, ""), strings.Join(nullParentInfo, ""))
			msoAlgorithm := ""
			if multiSiteOriginAlgorithm == 0 {
				msoAlgorithm = "consistent_hash"
				if ds.QStringIgnore != 0 {
					parentQstring = "consider"
				}
			} else if multiSiteOriginAlgorithm == 1 {
				msoAlgorithm = "false"
			} else if multiSiteOriginAlgorithm == 2 {
				msoAlgorithm = "strict"
			} else if multiSiteOriginAlgorithm == 3 {
				msoAlgorithm = "true"
			} else if multiSiteOriginAlgorithm == 4 {
				msoAlgorithm = "latched"
			} else {
				msoAlgorithm = "consistent_hash"
			}
			s += fmt.Sprintf("%s round_robin=%s qstring=%s go_direct=false parent_is_proxy=false\n", parents, msoAlgorithm, parentQstring)
		}

	}
	// TODO log
	// $self->app->log->debug( "MID PARENT.CONFIG:\n" . $text . "\n" );
	return s, nil
}

// uniqStrs removes duplicates from a sorted slice. This could be changed to modify in-place for efficiency, if necessary
func uniqStrs(strs []string) []string {
	uniq := []string{}
	if len(strs) == 0 {
		return uniq
	}
	for i := 1; i < len(strs); i++ {
		if strs[i] != strs[i-1] {
			uniq = append(uniq, strs[i])
		}
	}
	return uniq
}

func formatParentInfo(p ParentData) string {
	host := ""
	if p.UseIPAddr {
		host = p.IPAddr
	} else {
		host = fmt.Sprintf("%s.%s", p.Host, p.Domain)
	}
	return fmt.Sprintf("%s:%s|%s;", host, p.Port, p.Weight)
}

// sortParentsByRank sorts the given parents by rank. This mutates the given slice, and returns it. If the original slice must be preserved, pass a copy.
func sortParentsByRank(parents []ParentData) []ParentData {
	sort.Sort(ParentDatas(parents))
	return parents
}

// ParentDatas implements sort.Interface with Less() comparing  ParentData.Rank
type ParentDatas []ParentData

func (p ParentDatas) Len() int {
	return len(p)
}
func (p ParentDatas) Less(i, j int) bool {
	return p[i].Rank < p[j].Rank
}
func (p ParentDatas) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type ParentData struct {
	Host            string
	Port            string
	Domain          string
	Weight          string
	UseIPAddr       bool
	Rank            int
	IPAddr          string
	PrimaryParent   bool
	SecondaryParent bool
}

func originCacheGroups(toClient towrap.ITrafficOpsSession) ([]to.CacheGroup, error) {
	// TODO add TO endpoint to get cachegroups by type
	cacheGroups, err := toClient.CacheGroups()
	if err != nil {
		return nil, err
	}
	orgCacheGroups := []to.CacheGroup{}
	for _, cg := range cacheGroups {
		if cg.Type == "ORG_LOC" {
			orgCacheGroups = append(orgCacheGroups, cg)
		}
	}
	return orgCacheGroups, nil
}

func getCachegroup(toClient towrap.ITrafficOpsSession, serverHostName string) (to.CacheGroup, error) {
	server, err := getServer(toClient, serverHostName)
	if err != nil {
		return to.CacheGroup{}, fmt.Errorf("getting server %s: %v", serverHostName, err)
	}
	cacheGroup := server.Cachegroup
	cacheGroups, err := toClient.CacheGroups()
	for _, cg := range cacheGroups {
		if cg.Name == cacheGroup {
			return cg, nil
		}
	}
	return to.CacheGroup{}, fmt.Errorf("not found")
}

func strArrToMap(strs []string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, s := range strs {
		m[s] = struct{}{}
	}
	return m
}

func cacheGroupsNames(cgs []to.CacheGroup) []string {
	names := []string{}
	for _, cg := range cgs {
		names = append(names, cg.Name)
	}
	return names
}

// Returns map[foo]ParentData
// TODO change to take server object, for efficiency
func parentData(toClient towrap.ITrafficOpsSession, serverHostName string, paramMap map[string]map[string]string) (map[string][]ParentData, error) {
	serverTypeStr, err := getServerTypeStr(toClient, serverHostName)
	if err != nil {
		return nil, fmt.Errorf("error getting server '%v' type: %v", serverHostName, err)
	}

	parentCachegroups := []string{}
	secondaryParentCachegroups := []string{}
	if isMid := strings.HasPrefix(serverTypeStr, "MID"); isMid {
		if parentCachegroupsReal, err := originCacheGroups(toClient); err != nil {
			return nil, fmt.Errorf("getting cachegroups: %v", err)
		} else {
			parentCachegroups = cacheGroupsNames(parentCachegroupsReal)
		}
	} else {
		cacheGroup, err := getCachegroup(toClient, serverHostName)
		if err != nil {
			return nil, fmt.Errorf("getting %s cachegroup: %v", serverHostName, err)
		}
		parentCachegroups = append(parentCachegroups, cacheGroup.ParentName)
		if cacheGroup.SecondaryParentName != "" {
			secondaryParentCachegroups = append(secondaryParentCachegroups, cacheGroup.SecondaryParentName)
		}
	}

	serverDomain, err := getDomain(paramMap)
	if err != nil {
		return nil, fmt.Errorf("getting domain: %v", err)
	}

	serverCacheGroup, err := getCachegroup(toClient, serverHostName)
	if err != nil {
		return nil, fmt.Errorf("getting cachegroup for %s: %v", serverHostName, err)
	}

	allParentCachegroups := append(parentCachegroups, secondaryParentCachegroups...)

	// debug: domainServers = delivery_services, profileInfos = profile_cache
	profileCache, domainServers, err := getCachegroupsProfileInfo(strArrToMap(allParentCachegroups), toClient, paramMap)
	if err != nil {
		return nil, fmt.Errorf("getting cachegroups profile info: %v", err)
	}

	parentInfo := map[string][]ParentData{}
	for dsDomain, servers := range domainServers {
		for _, server := range servers {
			profileInfo := profileCache[server.Profile]
			if serverDomain != "" && profileInfo.Domain == serverDomain {
				port := profileInfo.Port
				if port == "" {
					port = strconv.Itoa(server.TCPPort)
				}

				primaryParent := serverCacheGroup.ParentName
				secondaryParent := serverCacheGroup.ParentName
				hasPrimaryParent := primaryParent == server.Cachegroup
				hasSecondaryParent := secondaryParent == server.Cachegroup

				parentInfo[dsDomain] = append(parentInfo[dsDomain], ParentData{
					Host:            server.HostName,
					Port:            port,
					Domain:          serverDomain,
					Weight:          profileInfo.Weight,
					UseIPAddr:       profileInfo.UseIPAddr,
					Rank:            profileInfo.Rank,
					IPAddr:          server.IPAddress,
					PrimaryParent:   hasPrimaryParent,
					SecondaryParent: hasSecondaryParent,
				})
			}
		}
	}
	return parentInfo, nil
}

func createParentDotConfigEdge(toClient towrap.ITrafficOpsSession, filename string, trafficOpsHost string, trafficServerHost string, params []to.Parameter) (string, error) {
	// "True" Parent
	paramMap := createParamsMap(params)
	pinfo, err := parentData(toClient, trafficServerHost, paramMap)
	if err != nil {
		return "", fmt.Errorf("getting parent data: %v", err)
	}

	deliveryServices, err := toClient.DeliveryServices()
	if err != nil {
		return "", fmt.Errorf("error getting delivery services: %v", err)
	}

	done := map[string]struct{}{} // map[originFQDN]

	s := "# DO NOT EDIT - Generated for " + trafficServerHost + " by Traffic Ops (" + trafficOpsHost + ") on " + time.Now().String() + "\n"

	for _, ds := range deliveryServices {
		org := ds.OrgServerFQDN
		parentQstring := "ignore"

		if org == "" {
			continue
		}

		if _, ok := done[org]; ok {
			continue
		}
		done[org] = struct{}{}

		orgUri, err := url.Parse(ds.OrgServerFQDN)
		if err != nil {
			return "", fmt.Errorf("parsing ds %v OrgServerFQDN '%s': %v", ds.XMLID, ds.OrgServerFQDN, err)
		}

		if ds.Type == "HTTP_NO_CACHE" || ds.Type == "HTTP_LIVE" || ds.Type == "DNS_LIVE" {
			s += fmt.Sprintf("dest_domain=%s port=%s go_direct=true\n", uriHost(orgUri), uriPort(orgUri))
			continue
		}

		if ds.QStringIgnore == 0 {
			parentQstring = "consider"
		}

		parentInfo := []string{}
		secondaryParentInfo := []string{}
		for _, parent := range pinfo["all_parents"] {
			if parent.PrimaryParent {
				parentInfo = append(parentInfo, formatParentInfo(parent))
			} else if parent.SecondaryParent {
				secondaryParentInfo = append(secondaryParentInfo, formatParentInfo(parent))
			}
		}
		parentInfo = uniqStrs(parentInfo)
		secondaryParentInfo = uniqStrs(secondaryParentInfo)

		parents := fmt.Sprintf(`parent="%s"`, strings.Join(parentInfo, ""))
		secParents := ""
		if len(secondaryParentInfo) > 0 {
			secParents = fmt.Sprintf(`secondary_parent="%s"`, strings.Join(secondaryParentInfo, ""))
		}
		roundRobin := "round_robin=consistent_hash"
		goDirect := "go_direct=false"
		s += fmt.Sprintf("dest_domain=%s port=%s %s %s %s %s qstring=%s\n", uriHost(orgUri), uriPort(orgUri), parents, secParents, roundRobin, goDirect, parentQstring)
	}

	pSelectAlg := getParamDefault(paramMap, "parent.config", "algorithm", "")
	if pSelectAlg == "consistent_hash" {
		parentInfo := []string{}
		for _, parent := range pinfo["all_parents"] {
			parentInfo = append(parentInfo, fmt.Sprintf("%s.%s:%s|%s;", parent.Host, parent.Domain, parent.Port, parent.Weight))
		}
		parentInfo = uniqStrs(parentInfo)
		s += fmt.Sprintf(`dest_domain=. parent="%s" round_robin=consistent_hash go_direct=false`, strings.Join(parentInfo, ""))
	} else { // default to old situation.
		parentInfo := []string{}
		for _, parent := range pinfo["all_parents"] {
			parentInfo = append(parentInfo, fmt.Sprintf("%s.%s:%s;", parent.Host, parent.Domain, parent.Port))
		}
		parentInfo = uniqStrs(parentInfo)
		s += fmt.Sprintf(`dest_domain=. parent="%s" round_robin=urlhash go_direct=false`, strings.Join(parentInfo, ""))
	}
	qstring := getParamDefault(paramMap, "parent.config", "qstring", "")
	if qstring != "" {
		s += fmt.Sprintf(" qstring=%s", qstring)
	}
	s += "\n"

	// $self->app->log->debug($text);
	return s, nil
}

// getCachegroupServers returns the servers which are ONLINE or REPORTED and in the given cachegroup
// TODO pass servers as param, for efficiency
func getCachegroupsOnlineServers(cacheGroups map[string]struct{}, toClient towrap.ITrafficOpsSession) ([]to.Server, error) {
	servers, err := toClient.Servers()
	if err != nil {
		return []to.Server{}, fmt.Errorf("error getting servers from Traffic Ops: %v", err)
	}
	cgServers := []to.Server{}
	for _, server := range servers {
		_, inCachegroups := cacheGroups[server.Cachegroup]
		if (server.Status == "REPORTED" || server.Status == "ONLINE") && inCachegroups {
			cgServers = append(cgServers, server)
		}
	}
	return cgServers, nil
}

type CachegroupProfileInfo struct {
	Domain    string // TODO remove?
	Weight    string
	Port      string
	UseIPAddr bool
	Rank      int
}

func deliveryServicesToMap(dses []to.DeliveryService) map[string]to.DeliveryService {
	m := map[string]to.DeliveryService{}
	for _, ds := range dses {
		m[ds.XMLID] = ds
	}
	return m
}

func getCachegroupsProfileInfo(cacheGroups map[string]struct{}, toClient towrap.ITrafficOpsSession, paramMap map[string]map[string]string) (map[string]CachegroupProfileInfo, map[string][]to.Server, error) {
	profileInfos := map[string]CachegroupProfileInfo{} // map[profileName]info
	domainServers := map[string][]to.Server{}          // map[deliveryServiceDomain]server

	serverDomain, err := getDomain(paramMap)
	if err != nil {
		return nil, nil, fmt.Errorf("getting domain: %v", err)
	}

	servers, err := getCachegroupsOnlineServers(cacheGroups, toClient)
	if err != nil {
		return nil, nil, fmt.Errorf("getting cachegroup servers: %v", err)
	}

	deliveryServices, err := toClient.DeliveryServices()
	if err != nil {
		return nil, nil, fmt.Errorf("error getting delivery services: %v", err)
	}
	deliveryServicesMap := deliveryServicesToMap(deliveryServices)

	for _, server := range servers {
		if server.Type != "ORG" && !strings.HasPrefix(server.Type, "EDGE") && !strings.HasPrefix(server.Type, "MID") {
			continue
		}
		if server.Type == "ORG" {
			serverDses, err := getServerDeliveryServices(toClient, server)
			if err != nil {
				return nil, nil, fmt.Errorf("getting cachegroup server %v delivery services: %v", server.HostName, err)
			}
			for dsName, _ := range serverDses {
				ds, ok := deliveryServicesMap[dsName]
				if !ok {
					return nil, nil, fmt.Errorf("delivery service %s in server but not deliveryservices", dsName)
				}
				dsDomain := stripProtocol(ds.OrgServerFQDN)
				slashPos := strings.Index(dsDomain, "/")
				if slashPos != -1 {
					dsDomain = dsDomain[:slashPos]
				}
				domainServers[dsDomain] = append(domainServers[dsDomain], server)
			}
		} else {
			domainServers["all_parents"] = append(domainServers["all_parents"], server)
		}

		// get the profile info, and cache it in profileInfos

		if _, ok := profileInfos[server.Profile]; !ok {
			// domain := getParam(paramMap map[string]map[string]string, file, param string) (string, bool) {

			profileInfos[server.Profile] = CachegroupProfileInfo{
				Domain:    serverDomain,
				Weight:    getParamDefault(paramMap, "parent.config", "weight", "0.999"),
				Port:      getParamDefault(paramMap, "parent.config", "port", ""),
				UseIPAddr: getParamDefaultCBool(paramMap, "parent.config", "use_ip_address", false),
				Rank:      getParamDefaultInt(paramMap, "parent.config", "rank", 1),
			}
		}

	}

	return profileInfos, domainServers, nil
}

func getParamDefault(paramMap map[string]map[string]string, file, name, defaultVal string) string {
	p, ok := getParam(paramMap, file, name)
	if !ok {
		return defaultVal
	}
	return p
}

func getParamDefaultCBool(paramMap map[string]map[string]string, file, name string, defaultVal bool) bool {
	p := getParamDefault(paramMap, file, name, "")
	if p == "" {
		return defaultVal
	}
	return p != "0"
}

func getParamDefaultInt(paramMap map[string]map[string]string, file, name string, defaultVal int) int {
	p := getParamDefault(paramMap, file, name, "")
	if i, err := strconv.Atoi(p); err != nil {
		return i
	}
	return defaultVal
}
