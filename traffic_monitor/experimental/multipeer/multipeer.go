package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

func main() {
	fmt.Println("Starting")
	toPort := 19999
	tmPortStart := 20000

	toPollInterval := time.Second * 3
	fakeTOHostName := "faketo0"

	fakeTO := NewFakeTrafficOps(toPort, fakeTOHostName)
	fakeTO.CRConfig = FakeCRConfig()
	fakeTO.Monitoring = FakeMonitoring()

	cdn := "cdn0"

	fakeTMs := map[tc.TrafficMonitorName]*FakeMonitor{}
	for i := 0; i < 10; i++ {
		// note you can't just add more to 10 here - to add more monitors, you'll have to modify the data in FakeMonitoring.
		fakeTMHostName := tc.TrafficMonitorName("tm" + strconv.Itoa(i))
		fakeTM := NewFakeTrafficMonitor(string(fakeTMHostName), tmPortStart+i, cdn, "http://localhost:"+strconv.Itoa(toPort), toPollInterval)
		fakeTMs[fakeTMHostName] = fakeTM
	}

	fakeTO.Start()

	multiPeerServerPort := 19998
	mps := NewMultiPeerServer(multiPeerServerPort, "multipeerserver", fakeTO, fakeTMs)
	go mps.Start()

	for tmName, tm := range fakeTMs {
		tm.Start()
		mps.M.Lock()
		mps.TMRunning[tmName] = true
		mps.M.Unlock()
		time.Sleep(time.Second * 5) // sleep between starts, to give each new monitor time to steal and rebalance things
	}

	for {
		time.Sleep(time.Hour)
	}
}

type MultiPeerServer struct {
	Server   *http.Server
	HostName string
	Port     int

	TO  *FakeTrafficOps
	TMs map[tc.TrafficMonitorName]*FakeMonitor

	TMRunning map[tc.TrafficMonitorName]bool
	M         sync.RWMutex
}

func NewMultiPeerServer(port int, hostName string, to *FakeTrafficOps, tms map[tc.TrafficMonitorName]*FakeMonitor) *MultiPeerServer {
	ms := &MultiPeerServer{
		HostName:  hostName,
		Port:      port,
		TO:        to,
		TMs:       tms,
		TMRunning: map[tc.TrafficMonitorName]bool{},
	}
	sv := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: ms.Handler(),
	}
	ms.Server = sv

	for tmName, _ := range tms {
		ms.TMRunning[tmName] = false
	}

	return ms
}

func (sv *MultiPeerServer) Start() {
	fmt.Printf("MPS serving %v\n", sv.Server.Addr)
	sv.Server.ListenAndServe()
}

func (sv *MultiPeerServer) Stop() {
	sv.Server.Shutdown(context.Background())
}

func (sv *MultiPeerServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { sv.Handle(w, req) })
}

func (sv *MultiPeerServer) Routes() map[string]func(w http.ResponseWriter, req *http.Request) {
	return map[string]func(w http.ResponseWriter, req *http.Request){
		"/api/servers":        sv.HandleServers,
		"/api/stop-monitor/":  sv.HandleStopMonitor,
		"/api/start-monitor/": sv.HandleStartMonitor,
	}
}

func (fm *MultiPeerServer) Handle(w http.ResponseWriter, req *http.Request) {
	req.Header.Set("server", "multipeerserver")
	routes := fm.Routes()
	path := strings.ToLower(strings.TrimRight(req.URL.Path, "/"))
	for routeName, routeF := range routes {
		if strings.HasPrefix(path, routeName) {
			routeF(w, req)
			return
		}
	}
	http.NotFound(w, req)
}

type MPSServerResp struct {
	TrafficOps      map[string]MPSServerRespServer                `json:"traffic_ops"`
	TrafficMonitors map[tc.TrafficMonitorName]MPSServerRespServer `json:"traffic_monitors"`
}

type MPSServerRespServer struct {
	Port    int  `json:"port"`
	Running bool `json:"running"`
}

func (fm *MultiPeerServer) HandleServers(w http.ResponseWriter, req *http.Request) {
	resp := MPSServerResp{TrafficOps: map[string]MPSServerRespServer{}, TrafficMonitors: map[tc.TrafficMonitorName]MPSServerRespServer{}}
	fm.M.RLock()
	resp.TrafficOps[fm.TO.HostName] = MPSServerRespServer{Port: fm.TO.Port, Running: true}
	for _, tm := range fm.TMs {
		resp.TrafficMonitors[tc.TrafficMonitorName(tm.HostName)] = MPSServerRespServer{Port: tm.Port, Running: fm.TMRunning[tc.TrafficMonitorName(tm.HostName)]}
	}
	fm.M.RUnlock()
	writeJSONNoLock(w, &resp)
}

func (fm *MultiPeerServer) HandleStopMonitor(w http.ResponseWriter, req *http.Request) {
	monitorName := tc.TrafficMonitorName(strings.TrimPrefix(req.URL.Path, "/api/stop-monitor/"))

	tm, ok := fm.TMs[tc.TrafficMonitorName(monitorName)]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(string(monitorName) + " not found"))
		return
	}

	fm.M.RLock()
	tmRunning := fm.TMRunning[monitorName]
	fm.M.RUnlock()

	if !tmRunning {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tm.Stop()

	fm.M.Lock()
	fm.TMRunning[monitorName] = false // TODO lock
	fm.M.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func (fm *MultiPeerServer) HandleStartMonitor(w http.ResponseWriter, req *http.Request) {
	monitorName := tc.TrafficMonitorName(strings.TrimPrefix(req.URL.Path, "/api/start-monitor/"))

	tm, ok := fm.TMs[monitorName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(string(monitorName) + " not found"))
		return
	}

	fm.M.RLock()
	tmRunning := fm.TMRunning[monitorName]
	fm.M.RUnlock()

	if tmRunning {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tm.Start()

	fm.M.Lock()
	fm.TMRunning[monitorName] = true // TODO lock
	fm.M.Unlock()

	w.WriteHeader(http.StatusNoContent)
}
