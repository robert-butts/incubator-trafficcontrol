/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing,
 software distributed under the License is distributed on an
 "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 KIND, either express or implied.  See the License for the
 specific language governing permissions and limitations
 under the License.
*/

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/crconfig"
	dsdata "github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/deliveryservicedata"
	tmenum "github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/enum"
	"github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/peer"
	to "github.com/apache/incubator-trafficcontrol/traffic_ops/client"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

const Version = "0.3"
const UserAgent = "traffic_map/" + Version

const ClientTimeout = time.Duration(10 * time.Second)

const (
	ContentTypeJSON       = "application/json"
	ContentTypeCSS        = "text/css"
	ContentTypePNG        = "image/png"
	ContentTypeJavascript = "application/javascript"
)

// CacheDuration is the length of time to cache data results (CRStates, CRConfig, etc). If a client requests a data object, and the last request happened less than this duration in the past, the last value is returned. This is live data, so Cache-Control doesn't really apply here, but we don't want to let clients kill our servers. Cached results should return an Age header.
const CacheDuration = time.Duration(24 * time.Hour)

// CachedResult is designed for caching with closing lambdas. It MAY NOT be copied. If you need to pass this to something, pass the pointer.
type CachedResult struct {
	data []byte
	time time.Time
	m    sync.RWMutex
}

func (r *CachedResult) Get() ([]byte, time.Time) {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.data, r.time
}

func (r *CachedResult) Set(d []byte, t time.Time) {
	r.m.Lock()
	defer r.m.Unlock()
	r.data = d
	r.time = t
}

func httpClient() http.Client {
	return http.Client{Timeout: ClientTimeout}
}

// func pingMonitors(monitors []string) error {
// 	client := httpClient()

// 	for _, monitor := range monitors {
// 		resp, err := client.Get(fmt.Sprintf("http://%v/api/version", monitor))
// 		if err != nil {
// 			return fmt.Errorf("monitor %v error %v", monitor, err)
// 		}
// 		resp.Body.Close()
// 		if resp.StatusCode != http.StatusOK {
// 			return fmt.Errorf("monitor %v bad code %v", monitor, resp.StatusCode)
// 		}
// 	}
// 	return nil
// }

type IndexPage struct {
	TileURL   string
	InfluxURL string
}

func readFileOrDie(filename string) []byte {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", filename, err)
		os.Exit(1)
	}
	return f
}

func main() {
	toURL := flag.String("to", "", "Traffic Ops URL")
	toUser := flag.String("toUser", "", "Traffic Ops user")
	toPass := flag.String("toPass", "", "Traffic Ops password")
	tileUrl := flag.String("tileurl", "", "Template URL of the map tile server")
	influxUrl := flag.String("influxurl", "", "InfluxDB URL")
	port := flag.Int("port", 80, "Port to serve on")
	flag.Parse()

	if *tileUrl == "" || *toURL == "" || *toUser == "" || *influxUrl == "" {
		fmt.Printf("Usage: traffic_map -to to.example.net -toUser bill -toPass thelizard -tileurl https://{s}.tile.example.net/{z}/{x}/{y}.png -port 80 -influxurl = http://trafficstats.example.net\n")
		os.Exit(1)
	}

	toInsecure := true
	toClient, err := to.LoginWithAgent(*toURL, *toUser, *toPass, toInsecure, UserAgent, false, ClientTimeout)
	if err != nil {
		fmt.Printf("Error connecting to Traffic Ops: %v\n", err)
		os.Exit(1)
	}

	// if err := pingMonitors(monitors); err != nil {
	// 	fmt.Printf("Error pinging monitors: %v\n", err)
	// 	os.Exit(1)
	// }

	indexTempl, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Printf("Error parsing index.html: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { handler(w, r, indexTempl, *tileUrl, *influxUrl) })
	http.HandleFunc("/api/1.2/servers.json", getHandleServers(toClient))
	http.HandleFunc("/api/1.2/cdns.json", getHandleCDNs(toClient))
	http.HandleFunc("/api/1.2/cachegroups.json", getHandleCachegroups(toClient))
	http.HandleFunc("/publish/CrStates", getHandleCRStates(toClient))
	http.HandleFunc("/publish/DsStats", getHandleDSStats(toClient))
	http.HandleFunc("/CRConfig-Snapshots/", getHandleCRConfig(toClient))
	http.HandleFunc("/cg-grey.png", fileHandler("cg-grey.png", ContentTypePNG))
	http.HandleFunc("/cg-orange.png", fileHandler("cg-orange.png", ContentTypePNG))
	http.HandleFunc("/cg-red.png", fileHandler("cg-red.png", ContentTypePNG))
	http.HandleFunc("/leaflet.css", fileHandler("leaflet.css", ContentTypeCSS))
	http.HandleFunc("/leaflet.js", fileHandler("leaflet.js", ContentTypeJavascript))
	http.HandleFunc("/traffic_map.js", fileHandler("traffic_map.js", ContentTypeJavascript))
	http.HandleFunc("/traffic_map.css", fileHandler("traffic_map.css", ContentTypeCSS))
	http.Handle("/font-awesome/", http.StripPrefix("/font-awesome", http.FileServer(http.Dir("./font-awesome"))))
	http.Handle("/awesome-markers/", http.StripPrefix("/awesome-markers", http.FileServer(http.Dir("./awesome-markers"))))
	http.HandleFunc("/leaflet.groupedlayercontrol.min.css", fileHandler("leaflet.groupedlayercontrol.min.css", ContentTypeCSS))
	http.HandleFunc("/leaflet.groupedlayercontrol.min.js", fileHandler("leaflet.groupedlayercontrol.min.js", ContentTypeJavascript))
	http.HandleFunc("/leaflet.groupedlayercontrol.min.js.map", fileHandler("leaflet.groupedlayercontrol.min.js.map", ContentTypeJavascript))
	http.HandleFunc("/us-states-geojson.min.json", fileHandler("us-states-geojson.min.json", ContentTypeJSON))
	http.HandleFunc("/zip-to-state-name.json", fileHandler("zip-to-state-name.json", ContentTypeJSON))
	http.HandleFunc("/us-counties-geojson.min.json", fileHandler("us-counties-geojson.min.json", ContentTypeJSON))
	http.HandleFunc("/us-state-county-zips.min.json", fileHandler("us-state-county-zips.min.json", ContentTypeJSON))

	fmt.Printf("Serving on %v\n", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		fmt.Printf("Error serving: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func fileHandler(filename, contentType string) http.HandlerFunc {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", filename, err)
		os.Exit(1)
	}
	return staticHandler(f, contentType)
}

func staticHandler(b []byte, contentType string) http.HandlerFunc {
	h := func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
		w.Header().Set("Content-Type", contentType)
		fmt.Fprintf(w, "%s", string(b))
	}
	return makeGzipHandler(h)
}

func handler(w http.ResponseWriter, r *http.Request, indexTempl *template.Template, tileURL string, influxURL string) {
	fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)

	dindexTempl, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Printf("%v %v %v error parsing index.html: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dindexTempl.Execute(w, IndexPage{TileURL: tileURL, InfluxURL: influxURL})
}

func getCRConfigs(toClient *to.Session) ([]crconfig.CRConfig, error) {
	crConfigs := []crconfig.CRConfig{}
	cdns, err := toClient.CDNs()
	if err != nil {
		return nil, fmt.Errorf("getting CDNs: %v", err)
	}

	for _, cdn := range cdns {
		crConfigBts, _, err := toClient.GetCRConfig(cdn.Name)
		if err != nil {
			return nil, fmt.Errorf("getting %v CRConfig: %v", cdn.Name, err)
		}
		crConfig := crconfig.CRConfig{}
		if err := json.Unmarshal(crConfigBts, crConfig); err != nil {
			return nil, fmt.Errorf("unmarshalling %v CRConfig: %v", cdn.Name, err)
		}
		crConfigs = append(crConfigs, crConfig)
	}
	return crConfigs, nil
}

type SimpleServer struct {
	HostName   string `json:"hostName"`
	Cachegroup string `json:"cachegroup"`
	CDNName    string `json:"cdnName"`
}

type SimpleServerResponse struct {
	Response []SimpleServer `json:"response"`
}

func serversToSimple(servers []to.Server) []SimpleServer {
	ss := make([]SimpleServer, 0, len(servers))
	for _, s := range servers {
		ss = append(ss, SimpleServer{
			HostName:   s.HostName,
			Cachegroup: s.Cachegroup,
			CDNName:    s.CDNName,
		})
	}
	return ss
}

func getHandleServers(toClient *to.Session) http.HandlerFunc {
	// TODO change use one CRConfig cache for all data that comes from it
	return makeCachedHandler(CacheDuration, ContentTypeJSON, func() ([]byte, error) {
		servers, err := toClient.ServersByType(url.Values{"type": []string{"EDGE"}})
		if err != nil {
			return nil, fmt.Errorf("error getting Servers: %v", err)
		}
		resp := SimpleServerResponse{Response: serversToSimple(servers)}
		bts, err := json.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("error marshalling Servers: %v", err)
		}
		return bts, nil
	})
}

func getHandleCDNs(toClient *to.Session) http.HandlerFunc {
	return makeCachedHandler(CacheDuration, ContentTypeJSON, func() ([]byte, error) {
		cdns, err := toClient.CDNs()
		if err != nil {
			return nil, fmt.Errorf("error getting CDNs: %v", err)
		}
		resp := to.CDNResponse{Response: cdns}
		bts, err := json.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("error marshalling CDNs: %v", err)
		}
		return bts, nil
	})
}

func getHandleCachegroups(toClient *to.Session) http.HandlerFunc {
	return makeCachedHandler(CacheDuration, ContentTypeJSON, func() ([]byte, error) {
		cachegroups, err := toClient.CacheGroups()
		if err != nil {
			return nil, fmt.Errorf("error getting Cachegroups: %v", err)
		}

		resp := to.CacheGroupResponse{Response: cachegroups}
		bts, err := json.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("error marshalling Cachegroups: %v", err)
		}
		return bts, nil
	})
}

func getCRStates(toClient *to.Session) (peer.Crstates, error) {
	states := peer.NewCrstates()

	monitors, err := toClient.ServersByType(url.Values{"type": []string{"RASCAL"}})
	if err != nil {
		return states, fmt.Errorf("getting servers: %v", err)
	}

	client := httpClient()
	seenCDNs := map[string]struct{}{}
	for _, monitor := range monitors {
		// TODO fix hardcoded statuses?
		if monitor.Status != "ONLINE" && monitor.Status != "REPORTED" {
			continue
		}

		if _, ok := seenCDNs[monitor.CDNName]; ok {
			continue // only check one monitor per CDN
		}
		seenCDNs[monitor.CDNName] = struct{}{}

		fqdn := monitor.HostName + "." + monitor.DomainName
		resp, err := client.Get("http://" + fqdn + "/publish/CrStates")
		if err != nil {
			return states, fmt.Errorf("getting %v CRStates: %v", monitor, err)
		}
		defer resp.Body.Close()

		crs := peer.Crstates{}
		if err := json.NewDecoder(resp.Body).Decode(&crs); err != nil {
			return states, fmt.Errorf("unmarshalling %v CRStates: %v", monitor, err)
		}

		for name, available := range crs.Caches {
			states.Caches[name] = available
		}
		for name, ds := range crs.Deliveryservice {
			states.Deliveryservice[name] = ds
		}
	}
	return states, nil
}

func getHandleCRStates(toClient *to.Session) http.HandlerFunc {
	return makeCachedHandler(CacheDuration, ContentTypeJSON, func() ([]byte, error) {
		crStates, err := getCRStates(toClient)
		if err != nil {
			return nil, fmt.Errorf("error getting CRStates: %v", err)
		}
		bts, err := json.Marshal(crStates)
		if err != nil {
			return nil, fmt.Errorf("error marshalling CRStates: %v", err)
		}
		return bts, nil
	})
}

func getDSStats(toClient *to.Session) (*dsdata.StatsOld, error) {
	dsStats := &dsdata.StatsOld{
		DeliveryService: map[tmenum.DeliveryServiceName]map[dsdata.StatName][]dsdata.StatOld{},
	}

	monitors, err := toClient.ServersByType(url.Values{"type": []string{"RASCAL"}})
	if err != nil {
		return dsStats, fmt.Errorf("getting servers: %v", err)
	}

	client := httpClient()
	seenCDNs := map[string]struct{}{}
	for _, monitor := range monitors {
		// TODO fix hardcoded statuses?
		if monitor.Status != "ONLINE" && monitor.Status != "REPORTED" {
			continue
		}

		if _, ok := seenCDNs[monitor.CDNName]; ok {
			continue // only check one monitor per CDN
		}
		seenCDNs[monitor.CDNName] = struct{}{}

		fqdn := monitor.HostName + "." + monitor.DomainName
		// TODO change to query string out unnecessary stats
		resp, err := client.Get("http://" + fqdn + "/publish/DsStats")
		if err != nil {
			return dsStats, fmt.Errorf("getting %v DSStats: %v", monitor, err)
		}
		defer resp.Body.Close()

		dsStatsCache := dsdata.StatsOld{}
		if err := json.NewDecoder(resp.Body).Decode(&dsStatsCache); err != nil {
			return dsStats, fmt.Errorf("unmarshalling %v DSStats: %v", monitor, err)
		}

		for name, stats := range dsStatsCache.DeliveryService {
			dsStats.DeliveryService[name] = stats
		}
	}
	return dsStats, nil
}

func getHandleDSStats(toClient *to.Session) http.HandlerFunc {
	return makeCachedHandler(CacheDuration, ContentTypeJSON, func() ([]byte, error) {
		s, err := getDSStats(toClient)
		if err != nil {
			return nil, fmt.Errorf("error getting DSStats: %v", err)
		}
		bts, err := json.Marshal(s)
		if err != nil {
			return nil, fmt.Errorf("error marshalling DSStats: %v", err)
		}
		return bts, nil
	})
}

func getPathPart(path string, n int) string {
	if len(path) == 0 {
		return ""
	}

	for i := 0; i < n; i++ {
		idx := strings.Index(path[1:], "/")
		if idx == -1 {
			return ""
		}
		path = path[idx+1:]
	}
	fmt.Println(path)
	if len(path) == 0 {
		return path
	}
	idx := strings.Index(path[1:], "/")
	if idx == -1 {
		return path[1:]
	}
	return path[1 : idx+1]
}

func getHandleCRConfig(toClient *to.Session) http.HandlerFunc {
	cache := map[tmenum.CDNName]*CachedResult{}
	return func(w http.ResponseWriter, r *http.Request) {
		cdn := getPathPart(r.URL.Path, 1)
		if cdn == "" {
			fmt.Printf("%v %v %v error: invalid CDN\n", time.Now(), r.RemoteAddr, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bts := []byte{}
		err := error(nil)
		fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
		if _, ok := cache[tmenum.CDNName(cdn)]; !ok {
			cache[tmenum.CDNName(cdn)] = &CachedResult{}
		}
		cacheData, cacheTime := cache[tmenum.CDNName(cdn)].Get()
		age := time.Now().Sub(cacheTime)
		if age < CacheDuration {
			bts = cacheData
			w.Header().Set("Age", fmt.Sprintf("%d", int(age.Seconds())))
		} else {
			bts, _, err = toClient.GetCRConfig(cdn)
			if err != nil {
				fmt.Printf("%v %v %v error: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache[tmenum.CDNName(cdn)].Set(bts, time.Now())
		}

		bts, err = gzipIfAccepts(r, w, bts)
		if err != nil {
			fmt.Printf("%v %v %v error gzipping: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", ContentTypeJSON)
		fmt.Fprintf(w, "%s", bts)
	}
}

func makeCachedHandler(cacheDuration time.Duration, contentType string, get func() ([]byte, error)) http.HandlerFunc {
	cache := CachedResult{}
	return func(w http.ResponseWriter, r *http.Request) {
		bts := []byte{}
		err := error(nil)
		fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
		cacheData, cacheTime := cache.Get()
		age := time.Now().Sub(cacheTime)
		if age < cacheDuration {
			bts = cacheData
			w.Header().Set("Age", fmt.Sprintf("%d", int(age.Seconds())))
		} else {
			bts, err = get()
			if err != nil {
				fmt.Printf("%v %v %v error: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache.Set(bts, time.Now())
		}

		bts, err = gzipIfAccepts(r, w, bts)
		if err != nil {
			fmt.Printf("%v %v %v error gzipping: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", contentType)
		fmt.Fprintf(w, "%s", bts)
	}
}

func stripAllWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

func acceptsGzip(r *http.Request) bool {
	encodingHeaders := r.Header["Accept-Encoding"] // headers are case-insensitive, but Go promises to Canonical-Case requests
	for _, encodingHeader := range encodingHeaders {
		encodingHeader = stripAllWhitespace(encodingHeader)
		encodings := strings.Split(encodingHeader, ",")
		for _, encoding := range encodings {
			if strings.ToLower(encoding) == "gzip" { // encoding is case-insensitive, per the RFC
				return true
			}
		}
	}
	return false
}

func gzipIfAccepts(r *http.Request, w http.ResponseWriter, b []byte) ([]byte, error) {
	if len(b) == 0 || !acceptsGzip(r) {
		return b, nil
	}

	w.Header().Set("Content-Encoding", "gzip")

	buf := bytes.Buffer{}
	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write(b); err != nil {
		return nil, fmt.Errorf("gzipping bytes: %v")
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer: %v")
	}

	return buf.Bytes(), nil
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func makeGzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}
