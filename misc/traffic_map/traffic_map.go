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
	"github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/peer"
	to "github.com/apache/incubator-trafficcontrol/traffic_ops/client"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
)

const Version = "0.2"
const UserAgent = "traffic_map/" + Version

const ClientTimeout = time.Duration(10 * time.Second)

// CacheDuration is the length of time to cache data results (CRStates, CRConfig, etc). If a client requests a data object, and the last request happened less than this duration in the past, the last value is returned. This is live data, so Cache-Control doesn't really apply here, but we don't want to let clients kill our servers. Cached results should return an Age header.
const CacheDuration = time.Duration(10 * time.Second)

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
	TileURL string
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
	port := flag.Int("port", 80, "Port to serve on")
	flag.Parse()

	if *tileUrl == "" || *toURL == "" || *toUser == "" {
		fmt.Printf("Usage: traffic_map -to to.example.net -toUser bill -toPass thelizard -tileurl https://{s}.tile.example.net/{z}/{x}/{y}.png -port 80\n")
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

	addFileHandler := func(path, filename, contentType string) {
		f, err := ioutil.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", filename, err)
			os.Exit(1)
		}
		http.HandleFunc(path, makeStaticHandler(f, contentType))
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { handler(w, r, indexTempl, *tileUrl) })
	http.HandleFunc("/api/1.2/servers.json", getHandleServers(toClient))
	http.HandleFunc("/api/1.2/cdns.json", getHandleCDNs(toClient))
	http.HandleFunc("/api/1.2/cachegroups.json", getHandleCachegroups(toClient))
	http.HandleFunc("/publish/CrStates", getHandleCRStates(toClient))
	addFileHandler("/cg-grey.png", "cg-grey.png", "image/png")
	addFileHandler("/cg-orange.png", "cg-orange.png", "image/png")
	addFileHandler("/cg-red.png", "cg-red.png", "image/png")
	addFileHandler("/leaflet.css", "leaflet.css", "text/css")
	addFileHandler("/leaflet.js", "leaflet.js", "application/javascript")
	addFileHandler("/traffic_map.js", "traffic_map.js", "application/javascript")

	fmt.Printf("Serving on %v\n", *port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		fmt.Printf("Error serving: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func makeStaticHandler(b []byte, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
		w.Header().Set("Content-Type", contentType)
		fmt.Fprintf(w, "%s", b)
	}
}

func handler(w http.ResponseWriter, r *http.Request, indexTempl *template.Template, tileURL string) {
	fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)

	dindexTempl, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Printf("%v %v %v error parsing index.html: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dindexTempl.Execute(w, IndexPage{TileURL: tileURL})
}

func getCRConfigs(toClient *to.Session) ([]crconfig.CRConfig, error) {
	crConfigs := []crconfig.CRConfig{}
	cdns, err := toClient.CDNs()
	if err != nil {
		return nil, fmt.Errorf("getting CDNs: %v", err)
	}

	for _, cdn := range cdns {
		crConfigBytes, _, err := toClient.GetCRConfig(cdn.Name)
		if err != nil {
			return nil, fmt.Errorf("getting %v CRConfig: %v", cdn.Name, err)
		}
		crConfig := crconfig.CRConfig{}
		if err := json.Unmarshal(crConfigBytes, crConfig); err != nil {
			return nil, fmt.Errorf("unmarshalling %v CRConfig: %v", cdn.Name, err)
		}
		crConfigs = append(crConfigs, crConfig)
	}
	return crConfigs, nil
}

func getHandleServers(toClient *to.Session) http.HandlerFunc {
	// TODO change use one CRConfig cache for all data that comes from it
	cache := CachedResult{}
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
		bytes := []byte{}
		cacheData, cacheTime := cache.Get()
		age := time.Now().Sub(cacheTime)
		if age < CacheDuration {
			bytes = cacheData
			w.Header().Set("Age", fmt.Sprintf("%d", int(age.Seconds())))
		} else {
			servers, err := toClient.Servers()
			if err != nil {
				fmt.Printf("%v %v %v error getting servers: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			resp := to.ServerResponse{Response: servers}
			bytes, err = json.Marshal(resp)
			if err != nil {
				fmt.Printf("%v %v %v error getting servers: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache.Set(bytes, time.Now())
		}
		bytes, err := gzipIfAccepts(r, w, bytes)
		if err != nil {
			fmt.Printf("%v %v %v error gzipping: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", bytes)
	}
}

func getHandleCDNs(toClient *to.Session) http.HandlerFunc {
	// TODO change use one CRConfig cache for all data that comes from it
	cache := CachedResult{}
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
		bytes := []byte{}
		cacheData, cacheTime := cache.Get()
		age := time.Now().Sub(cacheTime)
		if age < CacheDuration {
			bytes = cacheData
			w.Header().Set("Age", fmt.Sprintf("%d", int(age.Seconds())))
		} else {
			cdns, err := toClient.CDNs()
			if err != nil {
				fmt.Printf("%v %v %v error getting cdns: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			resp := to.CDNResponse{Response: cdns}
			bytes, err = json.Marshal(resp)
			if err != nil {
				fmt.Printf("%v %v %v error getting cdns: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache.Set(bytes, time.Now())
		}
		bytes, err := gzipIfAccepts(r, w, bytes)
		if err != nil {
			fmt.Printf("%v %v %v error gzipping: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", bytes)
	}
}

func getHandleCachegroups(toClient *to.Session) http.HandlerFunc {
	// TODO abstract cache logic, and put other code in "bytesGetter" type
	cache := CachedResult{}
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
		bytes := []byte{}
		cacheData, cacheTime := cache.Get()
		age := time.Now().Sub(cacheTime)
		if age < CacheDuration {
			bytes = cacheData
			w.Header().Set("Age", fmt.Sprintf("%d", int(age.Seconds())))
		} else {
			// crcs, err := getCRConfigs(monitors)
			cachegroups, err := toClient.CacheGroups()
			if err != nil {
				fmt.Printf("%v %v %v error getting cachegroups: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			resp := to.CacheGroupResponse{Response: cachegroups}
			bytes, err = json.Marshal(resp)
			if err != nil {
				fmt.Printf("%v %v %v error getting servers: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache.Set(bytes, time.Now())
		}

		bytes, err := gzipIfAccepts(r, w, bytes)
		if err != nil {
			fmt.Printf("%v %v %v error gzipping: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", bytes)
	}
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
	cache := CachedResult{}
	return func(w http.ResponseWriter, r *http.Request) {
		bytes := []byte{}
		fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
		cacheData, cacheTime := cache.Get()
		age := time.Now().Sub(cacheTime)
		if age < CacheDuration {
			bytes = cacheData
			w.Header().Set("Age", fmt.Sprintf("%d", int(age.Seconds())))
		} else {
			crs, err := getCRStates(toClient)
			if err != nil {
				fmt.Printf("%v %v %v error getting crstates: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			bytes, err = json.Marshal(crs)
			if err != nil {
				fmt.Printf("%v %v %v error marshalling crstates: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache.Set(bytes, time.Now())
		}
		bytes, err := gzipIfAccepts(r, w, bytes)
		if err != nil {
			fmt.Printf("%v %v %v error gzipping: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", bytes)
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
