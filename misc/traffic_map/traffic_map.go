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
	to "github.com/apache/incubator-trafficcontrol/traffic_ops/client"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
)

const Version = "0.1"
const UserAgent = "traffic_map/" + Version

var monitorsStr string
var port int
var tileUrl string

const ClientTimeout = time.Duration(10 * time.Second)

func httpClient() http.Client {
	return http.Client{Timeout: ClientTimeout}
}

func pingMonitors(monitors []string) error {
	client := httpClient()

	for _, monitor := range monitors {
		resp, err := client.Get(fmt.Sprintf("http://%v/api/version", monitor))
		if err != nil {
			return fmt.Errorf("monitor %v error %v", monitor, err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("monitor %v bad code %v", monitor, resp.StatusCode)
		}
	}
	return nil
}

type IndexPage struct {
	TileURL string
}

func main() {
	flag.StringVar(&monitorsStr, "monitors", "", "comma-separated list of Traffic Monitor FQDNs, recommended one from each CDN")
	flag.StringVar(&tileUrl, "tileurl", "", "Template URL of the map tile server")
	flag.IntVar(&port, "port", 80, "Port to serve on")
	flag.Parse()

	monitors := strings.Split(monitorsStr, ",")
	if tileUrl == "" || len(monitors) == 0 {
		fmt.Printf("Usage: traffic_map -monitors tm.cdn0.example.net,tm.cdn1.example.net -tileurl https://{s}.tile.example.net/{z}/{x}/{y}.png -port 80\n")
		return
	}

	if err := pingMonitors(monitors); err != nil {
		fmt.Printf("Error pinging monitors: %v\n", err)
		os.Exit(1)
	}

	// rawPage, err := ioutil.ReadFile("index.html")
	// if err != nil {
	// 	fmt.Printf("Error reading index.html: %v\n", err)
	// 	os.Exit(1)
	// }

	indexTempl, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Printf("Error parsing index.html: %v\n", err)
		os.Exit(1)
	}

	iconCgGrey, err := ioutil.ReadFile("cg-grey.png")
	if err != nil {
		fmt.Printf("Error reading cg-grey.png: %v\n", err)
		os.Exit(1)
	}
	iconCgOrange, err := ioutil.ReadFile("cg-orange.png")
	if err != nil {
		fmt.Printf("Error reading cg-orange.png: %v\n", err)
		os.Exit(1)
	}
	iconCgRed, err := ioutil.ReadFile("cg-red.png")
	if err != nil {
		fmt.Printf("Error reading cg-red.png: %v\n", err)
		os.Exit(1)
	}
	leafletCss, err := ioutil.ReadFile("leaflet.css")
	if err != nil {
		fmt.Printf("Error reading leaflet.css: %v\n", err)
		os.Exit(1)
	}
	leafletJs, err := ioutil.ReadFile("leaflet.js")
	if err != nil {
		fmt.Printf("Error reading leaflet.js: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { handler(w, r, indexTempl, tileUrl) })
	http.HandleFunc("/api/1.2/servers.json", func(w http.ResponseWriter, r *http.Request) { handleServers(w, r, monitors) })
	http.HandleFunc("/api/1.2/cachegroups.json", func(w http.ResponseWriter, r *http.Request) { handleCachegroups(w, r, monitors) })
	http.HandleFunc("/cg-grey.png", makeStaticHandler(iconCgGrey, "image/png"))
	http.HandleFunc("/cg-orange.png", makeStaticHandler(iconCgOrange, "image/png"))
	http.HandleFunc("/cg-red.png", makeStaticHandler(iconCgRed, "image/png"))
	http.HandleFunc("/leaflet.css", makeStaticHandler(leafletCss, "text/css"))
	http.HandleFunc("/leaflet.js", makeStaticHandler(leafletJs, "application/javascript"))

	fmt.Printf("Serving on %v\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
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

	dindexTempl.Execute(w, IndexPage{TileURL: tileUrl})
}

func getCRConfigs(monitors []string) ([]crconfig.CRConfig, error) {
	crcs := []crconfig.CRConfig{}
	client := httpClient()
	for _, monitor := range monitors {
		resp, err := client.Get(fmt.Sprintf("http://%v/publish/CrConfig", monitor))
		if err != nil {
			return nil, fmt.Errorf("getting %v CRConfig: %v", monitor, err)
		}
		defer resp.Body.Close()

		crc := crconfig.CRConfig{}
		if err := json.NewDecoder(resp.Body).Decode(&crc); err != nil {
			return nil, fmt.Errorf("unmarshalling %v CRConfig: %v", monitor, err)
		}
		crcs = append(crcs, crc)
	}
	return crcs, nil
}

type ServerResponse struct {
	Response []crconfig.Server `json:"response"`
}

func handleServers(w http.ResponseWriter, r *http.Request, monitors []string) {
	fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
	crcs, err := getCRConfigs(monitors)
	if err != nil {
		fmt.Printf("%v %v %v error getting servers: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	servers := []crconfig.Server{}
	for _, crc := range crcs {
		for _, server := range crc.ContentServers {
			servers = append(servers, server)
		}
	}

	resp := ServerResponse{Response: servers}
	bytes, err := json.Marshal(resp)
	if err != nil {
		fmt.Printf("%v %v %v error getting servers: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err = gzipIfAccepts(r, w, bytes)
	if err != nil {
		fmt.Printf("%v %v %v error gzipping: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", bytes)
}

func handleCachegroups(w http.ResponseWriter, r *http.Request, monitors []string) {
	fmt.Printf("%v serving %v %v\n", time.Now(), r.RemoteAddr, r.URL.Path)
	crcs, err := getCRConfigs(monitors)
	if err != nil {
		fmt.Printf("%v %v %v error getting cachegroups: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	cachegroups := []to.CacheGroup{}
	for _, crc := range crcs {
		for cgName, latlon := range crc.EdgeLocations {
			cg := to.CacheGroup{
				Name:      cgName,
				Latitude:  latlon.Lat,
				Longitude: latlon.Lon,
			}
			cachegroups = append(cachegroups, cg)
		}
	}

	resp := to.CacheGroupResponse{Response: cachegroups}
	bytes, err := json.Marshal(resp)
	if err != nil {
		fmt.Printf("%v %v %v error getting servers: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err = gzipIfAccepts(r, w, bytes)
	if err != nil {
		fmt.Printf("%v %v %v error gzipping: %v\n", time.Now(), r.RemoteAddr, r.URL.Path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", bytes)
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
