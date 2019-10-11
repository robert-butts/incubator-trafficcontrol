package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

type FakeTrafficOps struct {
	HostName string
	Port     int
	Server   *http.Server

	CRConfig   *tc.CRConfig
	Monitoring *TrafficMonitorConfig2
	M          sync.RWMutex // M locks all data. This is terribly inefficient, but for a fake app we don't care.
}

type TrafficMonitorConfig2 struct {
	tc.TrafficMonitorConfig
	MonitorPolledCacheGroups map[tc.TrafficMonitorName][]tc.CacheGroupName
}

func NewFakeTrafficOps(port int, hostName string) *FakeTrafficOps {
	fm := &FakeTrafficOps{HostName: hostName, Port: port}
	sv := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: fm.Handler(),
	}
	fm.Server = sv
	return fm
}

func (fm *FakeTrafficOps) Start() {
	go fm.Server.ListenAndServe()
}

func (fm *FakeTrafficOps) Stop() {
	fm.Server.Shutdown(context.Background())
}

func (fm *FakeTrafficOps) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { fm.Handle(w, req) })
}

const TOMonitorPolledCacheGroupsPath = `/monitor-polled-cachegroups`

func (fm *FakeTrafficOps) Routes() map[string]func(w http.ResponseWriter, req *http.Request) {
	return map[string]func(w http.ResponseWriter, req *http.Request){
		"/snapshot":                    fm.HandleCRConfig,
		"/monitoring.json":             fm.HandleMonitoring,
		TOMonitorPolledCacheGroupsPath: fm.HandleMonitorPolledCacheGroups,
	}
}

func (fm *FakeTrafficOps) Handle(w http.ResponseWriter, req *http.Request) {
	req.Header.Set("server", "fakeserver/"+fm.HostName)
	routes := fm.Routes()
	path := strings.ToLower(strings.TrimRight(req.URL.Path, "/"))
	for routeName, routeF := range routes {
		if strings.HasSuffix(path, routeName) {
			routeF(w, req)
			return
		}
	}
	http.NotFound(w, req)
}

func (fm *FakeTrafficOps) HandleCRConfig(w http.ResponseWriter, req *http.Request) {
	fm.writeJSON(w, req, &fm.CRConfig)
}

func (fm *FakeTrafficOps) HandleMonitoring(w http.ResponseWriter, req *http.Request) {
	// TODO wrap in response[]
	fm.writeJSON(w, req, &fm.Monitoring)
}

// writeJSON writes the given object as JSON.
// obj should be a member of fm, and MUST be a pointer. Passing a value will copy without locking.
func (fm *FakeTrafficOps) writeJSON(w http.ResponseWriter, req *http.Request, obj interface{}) {
	if req.Method != http.MethodGet {
		http.Error(w, "endpoint only supports GET method", http.StatusMethodNotAllowed)
		return
	}

	fm.M.RLock()
	bts, err := json.Marshal(obj)
	fm.M.RUnlock()
	if err != nil {
		http.Error(w, "error encoding json: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bts)
	w.Write([]byte("\n")) // makes the response a valid POSIX file, and still valid JSON.
}

// HandleMonitorPolledCacheGroups sets the POSTed monitors' cachegroups.
// Posted data adds to existing data, it doesn't replace. To unset a monitor's cachegroups,
// post that monitor with a nil or empty array.
func (fm *FakeTrafficOps) HandleMonitorPolledCacheGroups(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "endpoint only supports POST method", http.StatusMethodNotAllowed)
		return
	}

	newPolledCGs := map[tc.TrafficMonitorName][]tc.CacheGroupName{}
	if err := json.NewDecoder(req.Body).Decode(&newPolledCGs); err != nil {
		http.Error(w, "malformed json", http.StatusBadRequest)
		return
	}

	fm.M.Lock()
	if fm.Monitoring.MonitorPolledCacheGroups == nil {
		fm.Monitoring.MonitorPolledCacheGroups = map[tc.TrafficMonitorName][]tc.CacheGroupName{}
	}
	for monitor, cgs := range newPolledCGs {
		fm.Monitoring.MonitorPolledCacheGroups[monitor] = cgs
	}
	fm.M.Unlock()

	w.WriteHeader(http.StatusNoContent)
}
