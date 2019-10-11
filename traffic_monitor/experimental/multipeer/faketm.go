package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_monitor/cache"
	"github.com/apache/trafficcontrol/traffic_monitor/datareq"
	"github.com/apache/trafficcontrol/traffic_monitor/dsdata"
)

const MonitorPeerPollInterval = time.Second * 3 // TODO config?

type FakeMonitor struct {
	Server *http.Server

	HostName       string
	Port           int
	CDN            string
	TrafficOpsURI  string
	TOPollInterval time.Duration

	CRConfig   *tc.CRConfig
	Monitoring *TrafficMonitorConfig2

	CRStates     tc.CRStates
	DSStats      dsdata.Stats
	CacheStats   cache.Stats
	PeerStates   datareq.APIPeerStates
	PeersPolling map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{} // the list of peers this monitor is currently polling for cachegroup data it doesn't poll itself.
	// PolledCachegroups map[tc.CacheGroupName]struct{} // Note we don't keep a separate list, because we keep an up-to-date list of what this monitor is polling in Monitoring.MonitorPolledCacheGroups[HostName]

	PeerPollManager *MonitorPeerPollManager

	PeerData map[tc.TrafficMonitorName]MonitorPeerData

	TOPollDieChan chan struct{}

	M sync.RWMutex // M locks all data. This is terribly inefficient, but for a fake app we don't care.
}

func NewFakeTrafficMonitor(hostName string, port int, cdn string, toURI string, toPollInterval time.Duration) *FakeMonitor {
	fm := &FakeMonitor{
		HostName:       hostName,
		Port:           port,
		CDN:            cdn,
		TrafficOpsURI:  toURI,
		TOPollInterval: toPollInterval,

		CRConfig:   &tc.CRConfig{},
		Monitoring: &TrafficMonitorConfig2{},
		// PolledCachegroups: map[tc.CacheGroupName]struct{}{},
	}

	sv := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: fm.Handler(),
	}
	fm.Server = sv
	return fm
}

type MonitorPeerData struct {
	PolledCacheGroups []tc.CacheGroupName `json:"polled_cache_groups"`
}

// HandlePeerPollResults handles results from fm.PeerPollManager. Does not return.
func (fm *FakeMonitor) StartPeerPollResultsHandler() {
	for result := range fm.PeerPollManager.Results {
		// TODO create N processors, and pass the data to them, to not hold up all result processing on 1 thread.
		if result.Err != nil {
			// TODO find another monitor with these CGs to poll
			//      We need to wait at least the length of time of the Cache's poll (max of every cache in the group?)
			//      before considering the error real, and trying another monitor or polling ourself.
			//      To give a Monitor which just took a CG and announced it, time to actually poll it.
			//
			//      Alternatively: When taking a new CG, immediately poll every cache in it?
			//        Since Monitors refuse to serve until they've polled everything, that should be fine.
			fmt.Printf("ERROR POLL FakeMonitor.HandlePeerPollResults self '%v' polling '%v' err '%v'\n", fm.HostName, result.MonitorName, result.Err)
			continue
		}

		fm.M.Lock()
		fm.PeerData[result.MonitorName] = MonitorPeerData{PolledCacheGroups: []tc.CacheGroupName(result.Data)}
		fm.M.Unlock()
		fmt.Printf("INFO FakeMonitor.HandlePeerPollResults got result %+v\n", result)
	}
}

func (fm *FakeMonitor) StartTOPoll() {
	fm.TOPollDieChan = make(chan struct{})
	go func() {
		timer := time.NewTimer(fm.TOPollInterval)
		for {
			select {
			case <-fm.TOPollDieChan:
				timer.Stop()
				return
			case <-timer.C:
				start := time.Now()
				fm.PollTO()
				pollTime := time.Since(start)
				timer.Reset(fm.TOPollInterval - pollTime)
			}
		}
	}()
}

func (fm *FakeMonitor) Start() {
	fm.PeerPollManager = NewMonitorPeerPollManager(MonitorPeerPollInterval)
	go fm.StartPeerPollResultsHandler()
	// fmt.Println("INFO " + fm.HostName + "FakeMonitor.Start hostName '" + fm.HostName + "'")

	fm.PeersPolling = map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{}{}
	fm.PeerData = map[tc.TrafficMonitorName]MonitorPeerData{}

	// get and populate initial fake data
	fm.PollTO()
	fm.PopulateData()

	go fm.StartTOPoll()

	go fm.Server.ListenAndServe()
}

func (fm *FakeMonitor) Stop() {
	fm.TOPollDieChan <- struct{}{}
	fm.PeerPollManager.Stop() // closes the Results chan, stopping fm.HandlePEerPollResults
	fm.Server.Shutdown(context.Background())
}

func (fm *FakeMonitor) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) { fm.Handle(w, req) })
}

const PolledCacheGroupsPath = `/api/polled-cachegroups`
const PeerDataPath = `/api/peer-data`

func (fm *FakeMonitor) Routes() map[string]func(w http.ResponseWriter, req *http.Request) {
	return map[string]func(w http.ResponseWriter, req *http.Request){
		"/publish/crstates":   fm.HandleCrStates,
		"/publish/dsstats":    fm.HandleDsStats,
		"/publish/cachestats": fm.HandleCacheStats,
		"/publish/peerstates": fm.HandlePeerStates,

		// polled-cachegroups returns the cachegroups which this Monitor is polling
		PolledCacheGroupsPath: fm.HandlePolledCachegroups,
		// peer-data returns the data this monitor got from its peers
		PeerDataPath: fm.HandlePeerData,
		// TODO add endpoint to get which peers this monitor thinks have which CGs
	}
}

func (fm *FakeMonitor) Handle(w http.ResponseWriter, req *http.Request) {
	req.Header.Set("server", "fakemonitor/"+fm.HostName)
	routes := fm.Routes()
	path := strings.ToLower(strings.TrimRight(req.URL.Path, "/"))
	if route, ok := routes[path]; ok {
		route(w, req)
	} else {
		http.NotFound(w, req)
	}
}

func (fm *FakeMonitor) HandleCrStates(w http.ResponseWriter, req *http.Request) {
	fm.writeJSON(w, &fm.CRStates)
}

func (fm *FakeMonitor) HandleDsStats(w http.ResponseWriter, req *http.Request) {
	fm.writeJSON(w, &fm.DSStats)
}

func (fm *FakeMonitor) HandleCacheStats(w http.ResponseWriter, req *http.Request) {
	fm.writeJSON(w, &fm.CacheStats)
}

func (fm *FakeMonitor) HandlePeerStates(w http.ResponseWriter, req *http.Request) {
	fm.writeJSON(w, &fm.PeerStates)
}

// writeJSON writes the given object as JSON.
// obj should be a member of fm, and MUST be a pointer. Passing a value will copy without locking.
func (fm *FakeMonitor) writeJSON(w http.ResponseWriter, obj interface{}) {
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

// writeJSONNoLock writes the given object as JSON.
// This does not lock fm.M. This MUST NOT be used for shared data in a FakeMonitor. Use FakeMonitor.writeJSON instead.
func writeJSONNoLock(w http.ResponseWriter, obj interface{}) {
	bts, err := json.Marshal(obj)
	if err != nil {
		http.Error(w, "error encoding json: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(bts)
	w.Write([]byte("\n")) // makes the response a valid POSIX file, and still valid JSON.
}

// PopulateData populates the FakeMonitor data from the given CRConfig and TrafficMonitorConfig (monitoring.json).
// Data is populated with the Caches etc in the CRConfig and TrafficMonitorConfig, with randomized data values.
// PeerStates is not populated, as we expect to populate it from other FakeMonitors.
func (fm *FakeMonitor) PopulateData() {
	crs := tc.CRStates{Caches: map[tc.CacheName]tc.IsAvailable{}, DeliveryService: map[tc.DeliveryServiceName]tc.CRStatesDeliveryService{}}
	fm.M.RLock()
	for cacheName, _ := range fm.CRConfig.ContentServers {
		crs.Caches[tc.CacheName(cacheName)] = tc.IsAvailable{randBool()}
	}
	for dsName, _ := range fm.CRConfig.DeliveryServices {
		crs.DeliveryService[tc.DeliveryServiceName(dsName)] = tc.CRStatesDeliveryService{IsAvailable: randBool(), DisabledLocations: []tc.CacheGroupName{}}
	}
	fm.M.RUnlock()

	fm.M.Lock()
	fm.CRStates = crs
	fm.M.Unlock()
}

func randBool() bool { return rand.Int()%2 == 0 }

func (fm *FakeMonitor) PollTO() {
	resp, err := http.Get(fm.TrafficOpsURI + "/api/1.2/cdns/" + fm.CDN + "/snapshot")
	if err != nil {
		fmt.Println("ERROR " + fm.HostName + " FakeMonitor.PollTO: crconfig: get error: " + err.Error())
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR " + fm.HostName + " FakeMonitor.PollTO: crconfig: reading body: " + err.Error())
		return
	}
	crc := tc.CRConfig{}
	if err := json.Unmarshal(body, &crc); err != nil {
		fmt.Println("ERROR " + fm.HostName + " FakeMonitor.PollTO: crconfig: decoding response: " + err.Error() + "BB" + string(body) + "BB")
		return
	}

	resp, err = http.Get(fm.TrafficOpsURI + "/api/1.2/cdns/" + fm.CDN + "/configs/monitoring.json")
	if err != nil {
		fmt.Println("ERROR " + fm.HostName + " FakeMonitor.PollTO: monitoring: get error: " + err.Error())
		return
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("ERROR " + fm.HostName + " FakeMonitor.PollTO: monitoring: reading body: " + err.Error())
		return
	}

	mon := TrafficMonitorConfig2{}
	if err := json.Unmarshal(body, &mon); err != nil {
		fmt.Println("ERROR " + fm.HostName + " FakeMonitor.PollTO: monitoring: decoding response: " + err.Error())
		return
	}

	if mon.MonitorPolledCacheGroups == nil {
		mon.MonitorPolledCacheGroups = map[tc.TrafficMonitorName][]tc.CacheGroupName{}
	}

	fm.M.Lock()
	fm.CRConfig = &crc
	fm.Monitoring = &mon
	fm.M.Unlock()

	fm.RefreshPolledCacheGroups()
}

const MinStealCountThreshold = 1 // TODO make param - probably needs to be cdn-wide, monitors with different thresholds would cause flapping.

// MaxNearDistanceKM is the maximum distance for which a cachegroup is considered "near" to a monitor.
// 1000km is About a quarter width of the US.
// TODO make param - probably needs to be cdn-wide, monitors with different thresholds would cause flapping.
const MaxNearDistanceKM = 1000

func (fm *FakeMonitor) RefreshPolledCacheGroups() {
	allMonitorCoords := fm.GetAllMonitorCoordinates()

	// thisMonitorCoords, err := fm.Coordinates(allMonitorCoords)
	// if err != nil {
	// 	fmt.Println("ERROR getting coordinate: " + err.Error())
	// 	return
	// }

	// monitorCGNear := fm.GetCacheGroupNearness(allMonitorCoords)

	// // DEBUG
	// fmt.Printf("Nearness:\n")
	// for mName, cgNear := range monitorCGNear {
	// 	for cgName, cgIsNear := range cgNear {
	// 		fmt.Printf("monitor '%v' cg '%v' near %v\n", mName, cgName, cgIsNear)
	// 	}
	// }
	// fmt.Printf("\n")

	fm.M.RLock()
	monitorPolledCacheGroups := fm.Monitoring.MonitorPolledCacheGroups // map[tc.TrafficMonitorName][]tc.CacheGroupName
	fm.M.RUnlock()

	fm.M.RLock()
	hostName := tc.TrafficMonitorName(fm.HostName)
	fm.M.RUnlock()

	// newMonitorCGs is the changes we're going to make to the polled cachegroups in TO
	// This includes changes to ourself, and the removal of any offline monitors.
	// This WILL NOT include changes to online or reported monitors!
	// We MUST only change ourselves, or cleanup offlined monitors..
	newMonitorCGs := map[tc.TrafficMonitorName][]tc.CacheGroupName{}
	newMonitorCGs[hostName] = []tc.CacheGroupName{} // make an entry for this monitor, in case all we do is delete but not add, so it still POSTs.

	monitorsToRemove := fm.GetOfflineMonitorsToRemove()

	changedPolledCacheGroups := len(monitorsToRemove) > 0 // whether we changed anything, so we don't do an unnecessary POST to Traffic Ops.

	for _, monitor := range monitorsToRemove {
		newMonitorCGs[monitor] = []tc.CacheGroupName{}
	}

	monitorCGNear := fm.GetCacheGroupNearness(allMonitorCoords)

	equidistantMonitors, err := fm.GetEquidistantMonitors(monitorCGNear) // map[tc.CacheGroupName][]tc.TrafficMonitorName
	if err != nil {
		fmt.Println("ERROR " + fm.HostName + " getting equidistant monitors: " + err.Error())
		return
	}

	// We want to know, is there an equidistant monitor already polling the CG,
	// and if so, does that monitor poll <MinStealCountThreshold more CGs than this monitor.
	// If either of those are untrue, add that CG to this monitor.

	cacheGroups := []tc.CacheGroupName{} // EdgeLocations    map[string]CRConfigLatitudeLongitude `json:"edgeLocations,omitempty"`
	fm.M.RLock()
	for cgName, _ := range fm.CRConfig.EdgeLocations {
		cacheGroups = append(cacheGroups, tc.CacheGroupName(cgName))
	}
	fm.M.RUnlock()

	monitorPolledCacheGroupsMap := map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{}{}
	for monitorName, monitorCGs := range monitorPolledCacheGroups {
		monitorPolledCacheGroupsMap[monitorName] = map[tc.CacheGroupName]struct{}{}
		for _, cg := range monitorCGs {
			monitorPolledCacheGroupsMap[monitorName][cg] = struct{}{}
		}
	}
	// add this monitor to the map if there weren't any entries, to prevent nil panics later.
	if monitorPolledCacheGroupsMap[hostName] == nil {
		monitorPolledCacheGroupsMap[hostName] = map[tc.CacheGroupName]struct{}{}
	}

cgloop:
	for _, cgName := range cacheGroups {
		cgEquidistantMonitors := equidistantMonitors[cgName]
		for _, monitorName := range cgEquidistantMonitors {
			if monitorName == hostName {
				continue // skip self
			}

			// at this point, we skipped ourselves.

			if _, ok := monitorPolledCacheGroupsMap[monitorName][cgName]; !ok {
				// fmt.Printf("INFO "+fm.HostName+" cg '%v' not polled by equidistant monitor '%v', continuing looking for an equidistant monitor that is\n", cgName, monitorName)
				continue // skip monitors not polling this cg
			}

			// at this point, we skipped monitors not polling the cg.

			if len(monitorPolledCacheGroupsMap[monitorName])-len(monitorPolledCacheGroupsMap[hostName]) < MinStealCountThreshold {
				// fmt.Printf("INFO "+fm.HostName+" for cg '%v' self '%v' other '%v' len(%v)-len(%v) < %v\n", cgName, hostName, monitorName, monitorPolledCacheGroups[monitorName], monitorPolledCacheGroups[hostName], MinStealCountThreshold)

				// some monitor exists which is equidistant to this monitor, and not polling MinStealCountThreshold more than this monitor, so don't start polling this cachegroup.
				continue cgloop
			}

			// at this point, monitorName is polling the CG, but it has more than MinStealCountThreshold we skipped monitors not polling the cg.

			fmt.Printf("INFO "+fm.HostName+" stealing cg '%v' from %v because it has %v and we only have %v CGs\n", cgName, monitorName, len(monitorPolledCacheGroupsMap[monitorName]), len(monitorPolledCacheGroupsMap[hostName]))

			// Add the CG we're about to steal to the map as if we're already polling it,
			// and remove it from the other  monitor as if it already stopped polling it.
			// This is necessary, so we don't steal in excess of MinStealCountThreshold.
			monitorPolledCacheGroupsMap[hostName][cgName] = struct{}{}
			delete(monitorPolledCacheGroupsMap[monitorName], cgName)
		}

		// fmt.Printf("INFO "+fm.HostName+" cg '%v' not polled by any equidistant monitor with < minsteal, adding to this '%v'\n", cgName, hostName)

		// at this point, all the equidistant monitors polling this cachegroup have more than MinStealCountThreshold
		// cachegroups more than this monitor (possibly because no equidistant monitors exist),
		// so start this monitor polling the cachegroup
		newMonitorCGs[hostName] = append(newMonitorCGs[hostName], cgName)
	}

	// if we didn't already change cachegroups with removing offline monitors above,
	// check to see if we changed the cachegroups of this monitor.
	oldMonitorPolledCGs := monitorPolledCacheGroups[hostName]
	newMonitorPolledCGs := newMonitorCGs[hostName]
	// TODO make faster, if it matters?
	if !changedPolledCacheGroups {
		oldMonitorPolledCGMap := map[tc.CacheGroupName]struct{}{}
		for _, cg := range oldMonitorPolledCGs {
			oldMonitorPolledCGMap[cg] = struct{}{}
		}
		for _, cg := range newMonitorPolledCGs {
			if _, ok := oldMonitorPolledCGMap[cg]; !ok {

				fmt.Printf("INFO "+fm.HostName+" CHANGE ADD '%v'\n", cg)

				// we added a cg that wasn't previously polled, so there was a change.
				changedPolledCacheGroups = true
				// DEBUG - don't break, so we print every change
				// break
			}
		}
	}
	if !changedPolledCacheGroups {
		newMonitorPolledCGMap := map[tc.CacheGroupName]struct{}{}
		for _, cg := range newMonitorPolledCGs {
			newMonitorPolledCGMap[cg] = struct{}{}
		}
		for _, cg := range oldMonitorPolledCGs {
			if _, ok := newMonitorPolledCGMap[cg]; !ok {

				fmt.Printf("INFO "+fm.HostName+" CHANGE SUB '%v'\n", cg)

				// we removed a cg that was previously polled, so there was a change.
				changedPolledCacheGroups = true
				// DEBUG - don't break, so we print every change
				// break
			}
		}
	}

	if changedPolledCacheGroups {
		fmt.Printf("INFO "+fm.HostName+" posting polled cachegroups to Traffic Ops: %+v\n", newMonitorCGs)
		if err := fm.PostNewPolledCachegroups(newMonitorCGs); err != nil {
			fmt.Println("ERROR " + fm.HostName + " posting cachegroups to Traffic Ops: " + err.Error())
			return
		}

		//	newMonitorCGs := map[tc.TrafficMonitorName][]tc.CacheGroupName{}
		// manually update our copy, rather than asking TO again. We'll get TO again in the next poll.
		// TODO verify this is right. We don't want to hit TO too much, but at the same time, it would be
		//      good to make sure it didn't change underneath us. Or is it ok to wait until the next TO poll?
		fm.M.Lock()
		for monitorName, monitorCGs := range newMonitorCGs {
			fm.Monitoring.MonitorPolledCacheGroups[monitorName] = monitorCGs
		}
		fm.M.Unlock()
	} else {
		fmt.Printf("INFO " + fm.HostName + " no changes to polled cachegroups\n")
	}

	fm.RefreshPolledPeers()
}

// RefreshPolledPeers refreshes which peers we poll.
//
// We want to only poll 1 peer peer CacheGroup.
// We also want to poll as few peers as possible, i.e. we want to poll peers with lots of CGs.
// This is the Knapsack problem, and is NP-Complete to do perfectly.
// Therefore, we use the simple Greedy heuristic: poll the peers with the most CGs first.
//
// TODO also start peer poller, on Start()
// TODO only get monitors that are equidistant with fm from the cg. This will allow fm
//      to discover when an equidistant monitor isn't responding, and take up it's CGs.
//
func (fm *FakeMonitor) RefreshPolledPeers() {
	hostName := tc.TrafficMonitorName("")
	monitorCGs := map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{}{}

	// Copy the monitorCGs, to avoid repeated locking and changes underneath us

	fm.M.RLock()
	hostName = tc.TrafficMonitorName(fm.HostName)
	for monitorName, monitorPolledCGs := range fm.Monitoring.MonitorPolledCacheGroups {
		monitorCGs[monitorName] = map[tc.CacheGroupName]struct{}{}
		for _, cg := range monitorPolledCGs {
			monitorCGs[monitorName][cg] = struct{}{}
		}
	}
	fm.M.RUnlock()

	// Remove our own cachegroups from the poll counts.
	//   - we don't want to pick a Monitor that polls lots of CGs, if it mostly polls the same CGs as us.

	thisMonitorCGs := monitorCGs[hostName]
	delete(monitorCGs, hostName) // TODO rename monitorCGs -> otherMonitorCGs
	for cg, _ := range thisMonitorCGs {
		for monitorName, _ := range monitorCGs {
			if monitorName == hostName {
				continue
			}
			delete(monitorCGs[monitorName], cg)
		}
	}

	fmt.Printf("DEBUGGGG %v thisMonitorCGs %+v\n", hostName, thisMonitorCGs)
	fmt.Printf("DEBUGGGG %v post-self-removal monitorCGs %+v\n", hostName, monitorCGs)

	// Make an array of monitors, sorted by most cachegroups polled.

	monitorCGSort := MonitorCGs{}
	for monitor, cgs := range monitorCGs {
		monitorCGSort = append(monitorCGSort, MonitorCGCount{MonitorName: monitor, CGCount: len(cgs)})
	}
	sort.Sort(monitorCGSort)
	monitorsByCG := []tc.TrafficMonitorName{}
	for _, mo := range monitorCGSort {
		monitorsByCG = append(monitorsByCG, mo.MonitorName)
	}

	// Make the list of CGs we're not polling, that need polled.

	cgsToPoll := map[tc.CacheGroupName]struct{}{}
	for _, cgs := range monitorCGs {
		for cg, _ := range cgs {
			cgsToPoll[cg] = struct{}{}
		}
	}

	// Copy cgsToPoll, so we can keep the original list,
	// while removing from it as we eliminate CGs we need to add.
	cgsLeftToPoll := map[tc.CacheGroupName]struct{}{}
	for cg, _ := range cgsToPoll {
		cgsLeftToPoll[cg] = struct{}{}
	}

	// Make the list of monitors to poll, mapped to their expected cachegroups

	monitorsToPoll := map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{}{}
	for len(cgsLeftToPoll) > 0 {
		cg := tc.CacheGroupName("")
		for cgl, _ := range cgsLeftToPoll {
			cg = cgl // only way to get a random map element is to range. Thus Saith The Pike.
			break
		}
		cgMonitor := tc.TrafficMonitorName("")
		for _, monitorName := range monitorsByCG {
			if monitorName == hostName {
				continue // skip self
			}
			_, ok := monitorCGs[monitorName][cg]
			if !ok {
				continue // monitorName isn't polling cg, keep looking for one that is
			}
			cgMonitor = monitorName
			break
		}

		if cgMonitor == "" {
			fmt.Println("ERROR cg '" + string(cg) + "' unpolled when refreshing polled peers! This should never happen, because refreshing polled CGs should happen immediately before refreshing peers!")
			delete(cgsLeftToPoll, cg) // it isn't polled, so stop looking. We logged an error, and the next CG refresh should pick it up.
			continue
		}

		// Add this monitor with and any CGs it has in cgsLeftToPoll to monitorsToPoll,
		// and remove all it's CGs from cgsLeftToPoll.
		monitorsToPoll[cgMonitor] = map[tc.CacheGroupName]struct{}{}
		for monitorNameCG, _ := range monitorCGs[cgMonitor] {
			if _, ok := cgsLeftToPoll[monitorNameCG]; ok {
				monitorsToPoll[cgMonitor][monitorNameCG] = struct{}{}
				delete(cgsLeftToPoll, monitorNameCG)
			}

		}
	}

	fm.M.Lock()
	fm.PeersPolling = monitorsToPoll
	fm.M.Unlock()

	fm.M.RLock()
	monitorMap := map[tc.TrafficMonitorName]tc.TrafficMonitor{}
	for _, monitor := range fm.Monitoring.TrafficMonitors {
		monitorMap[tc.TrafficMonitorName(monitor.HostName)] = monitor
	}
	fm.M.RUnlock()

	pollData := []MonitorPeerPollData{}
	//monitorsToPoll := map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{}{}
	fm.M.RLock()
	for monitorName, _ := range monitorsToPoll {
		monitor, ok := monitorMap[monitorName]
		if !ok {
			fmt.Println("ERROR: monitor to poll not in Monitoring! Should never happen!")
			continue
		}
		pollData = append(pollData, MonitorPeerPollData{Monitor: monitor})
	}
	fm.M.RUnlock()

	fm.PeerPollManager.RefreshPollers(pollData, hostName)
}

type MonitorCGCount struct {
	MonitorName tc.TrafficMonitorName
	CGCount     int
}

// MonitorCGs implements sort.Interface
type MonitorCGs []MonitorCGCount

func (mc MonitorCGs) Len() int           { return len(mc) }
func (mc MonitorCGs) Less(i, j int) bool { return mc[i].CGCount < mc[j].CGCount }
func (mc MonitorCGs) Swap(i, j int)      { mc[i], mc[j] = mc[j], mc[i] }

// GetEquidistantMonitors returns, for each cachegroup, the list of monitors which are equidistant with this monitor.
// That is, if this monitor is "near" the cachegroup, all other "near" monitors; likewise with "far."
func (fm *FakeMonitor) GetEquidistantMonitors(monitorCGNear map[tc.TrafficMonitorName]map[tc.CacheGroupName]bool) (map[tc.CacheGroupName][]tc.TrafficMonitorName, error) {
	fm.M.RLock()
	hostName := tc.TrafficMonitorName(fm.HostName)
	fm.M.RUnlock()

	thisMonitorNear, ok := monitorCGNear[hostName]
	if !ok {
		return nil, errors.New("this monitor '" + string(hostName) + "' not in given monitor cachegroup nearness")
	}

	equidistantMonitors := map[tc.CacheGroupName][]tc.TrafficMonitorName{}

	for monitorName, monitorCGNears := range monitorCGNear {
		for cg, near := range monitorCGNears {
			thisCGNear, ok := thisMonitorNear[cg]
			if !ok {
				fmt.Println("ERROR " + fm.HostName + " monitor '" + string(monitorName) + "' has cachegroup '" + string(cg) + "' but this monitor '" + string(hostName) + "' doesn't! Not adding to equidistant monitors!")
				continue
			}
			if thisCGNear == near {
				equidistantMonitors[cg] = append(equidistantMonitors[cg], monitorName)
			}
		}
	}
	return equidistantMonitors, nil
}

// GetOfflineMonitorsToRemove returns the list of monitors with a cachegroup polled in TO, which are OFFLINE or ADMIN_DOWN, which should be removed from TO.
func (fm *FakeMonitor) GetOfflineMonitorsToRemove() []tc.TrafficMonitorName {
	fm.M.RLock()
	monitorPolledCacheGroups := fm.Monitoring.MonitorPolledCacheGroups
	crConfigMonitors := fm.CRConfig.Monitors
	fm.M.RUnlock()

	monitorsToRemove := []tc.TrafficMonitorName{}
	for monitorName, _ := range monitorPolledCacheGroups {
		crcMonitor, ok := crConfigMonitors[string(monitorName)]
		if !ok {
			fmt.Println("WARN " + fm.HostName + " TO polled monitor '" + string(monitorName) + "' not in CRConfig! Removing from TO polling!")
			monitorsToRemove = append(monitorsToRemove, monitorName)
			continue
		}
		if crcMonitor.ServerStatus == nil {
			fmt.Println("WARN " + fm.HostName + " TO polled monitor '" + string(monitorName) + "' CRConfig missing status! Leaving in!")
			continue
		}
		if *crcMonitor.ServerStatus == tc.CRConfigServerStatus(tc.CacheStatusOffline) || *crcMonitor.ServerStatus == tc.CRConfigServerStatus((tc.CacheStatusAdminDown)) {
			monitorsToRemove = append(monitorsToRemove, monitorName)
			continue
		}
	}
	return monitorsToRemove
}

// PostNewPolledCachegroups makes a POST to Traffic Ops with the new monitor-polled-cachegroups.
// Note entries in monitorCGs with nil or empty slices will remove all cachegroups from that monitor!
// May be given no new/changed monitors, in which case it will immediately return a nil error without contacting Traffic Ops.
func (fm *FakeMonitor) PostNewPolledCachegroups(monitorCGs map[tc.TrafficMonitorName][]tc.CacheGroupName) error {
	if len(monitorCGs) == 0 {
		return nil
	}

	reqURI := fm.TrafficOpsURI + "/api/1.2/cdns/" + fm.CDN + TOMonitorPolledCacheGroupsPath

	jsonBts, err := json.Marshal(&monitorCGs)
	if err != nil {
		return errors.New("encoding monitor cachegroups: " + err.Error())
	}

	resp, err := http.Post(reqURI, tc.ApplicationJson, bytes.NewBuffer(jsonBts))
	if err != nil {
		return errors.New("posting monitor cachegroups to Traffic Ops: " + err.Error())
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return errors.New("unexpected status: " + resp.Status)
	}
	return nil
}

func (fm *FakeMonitor) GetCacheGroupNearness(allMonitorCoords map[tc.TrafficMonitorName]tc.MonitoringCoordinates) map[tc.TrafficMonitorName]map[tc.CacheGroupName]bool {
	cgs := map[tc.CacheGroupName]tc.CRConfigLatitudeLongitude{}
	fm.M.RLock()
	for cgName, cgCoords := range fm.CRConfig.EdgeLocations {
		cgs[tc.CacheGroupName(cgName)] = cgCoords
	}
	fm.M.RUnlock()

	// TODO time

	monitorCGNear := map[tc.TrafficMonitorName]map[tc.CacheGroupName]bool{}
	for monitorName, monitorCoords := range allMonitorCoords {
		monitorCGNear[monitorName] = map[tc.CacheGroupName]bool{}
		for cgName, cgCoords := range cgs {
			distKM := getDistanceFromLatLonInKm(monitorCoords.Latitude, monitorCoords.Longitude, cgCoords.Lat, cgCoords.Lon)

			// fmt.Printf("INFO "+fm.HostName+" monitor '%v' cg '%v' dist %dkm near %v\n", monitorName, cgName, uint64(distKM), distKM < MaxNearDistanceKM)

			monitorCGNear[monitorName][cgName] = distKM < MaxNearDistanceKM
		}
	}

	return monitorCGNear
}

func (fm *FakeMonitor) GetAllMonitorCoordinates() map[tc.TrafficMonitorName]tc.MonitoringCoordinates {
	monitorCGs := map[tc.TrafficMonitorName]tc.CacheGroupName{}
	fm.M.RLock()
	for monitorName, monitorServer := range fm.CRConfig.Monitors {
		if monitorServer.Location == nil {
			fmt.Println("ERROR " + fm.HostName + " '" + monitorName + "' in CRConfig has nil location (cachegroup) - skipping!")
			continue
		}
		monitorCGs[tc.TrafficMonitorName(monitorName)] = tc.CacheGroupName(*monitorServer.Location)
	}
	fm.M.RUnlock()

	cgCoords := map[tc.CacheGroupName]tc.MonitoringCoordinates{}
	fm.M.RLock()
	for _, cgCoord := range fm.Monitoring.CacheGroups {
		// if cg.Coordinates == nil {
		// 	fmt.Println("monitor '" + monitorName + "' in monitoring has nil coordinates (cachegroup) - skipping!")
		// 	continue
		// }
		cgCoords[tc.CacheGroupName(cgCoord.Name)] = cgCoord.Coordinates
	}
	fm.M.RUnlock()

	coords := map[tc.TrafficMonitorName]tc.MonitoringCoordinates{}
	for monitorName, monitorCG := range monitorCGs {
		cgCoord, ok := cgCoords[monitorCG]
		if !ok {
			fmt.Println("ERROR " + fm.HostName + " monitor '" + string(monitorName) + "' CRConfig cachegroup (location) '" + string(monitorCG) + "' not found in monitoring - skipping!")
			// fmt.Printf("%+v\n", fm.Monitoring)
			continue
		}
		coords[monitorName] = cgCoord
	}

	return coords
}

func (fm *FakeMonitor) Coordinates(allMonitorCoords map[tc.TrafficMonitorName]tc.MonitoringCoordinates) (tc.MonitoringCoordinates, error) {
	hostName := tc.TrafficMonitorName("")
	fm.M.RLock()
	hostName = tc.TrafficMonitorName(fm.HostName)
	fm.M.RUnlock()
	coords, ok := allMonitorCoords[hostName]
	if !ok {
		return tc.MonitoringCoordinates{}, errors.New("self '" + string(hostName) + "' not found in CRConfig - can't continue!")
	}
	return coords, nil
}

func (fm *FakeMonitor) HandlePolledCachegroups(w http.ResponseWriter, req *http.Request) {
	cgArr := []tc.CacheGroupName{}
	fm.M.RLock()
	for _, cg := range fm.Monitoring.MonitorPolledCacheGroups[tc.TrafficMonitorName(fm.HostName)] {
		cgArr = append(cgArr, cg)
	}
	fm.M.RUnlock()
	writeJSONNoLock(w, cgArr)
}

func (fm *FakeMonitor) HandlePeerData(w http.ResponseWriter, req *http.Request) {
	// must copy data before writing, shared by multiple threads
	peerData := map[tc.TrafficMonitorName]MonitorPeerData{}
	fm.M.RLock()
	for mName, mData := range fm.PeerData {
		peerData[mName] = mData
	}
	fm.M.RUnlock()
	writeJSONNoLock(w, peerData)
}

// func (fm *FakeMonitor) PollPeersForPolledCacheGroups() {
// 	cgs := map[tc.CacheGroupName]tc.CRConfigLatitudeLongitude{}
// 	fm.M.RLock() // TODO something faster
// 	for cg, cgLatLon := range fm.CRConfig.EdgeLocations {
// 		cgs[tc.CacheGroupName(cg)] = cgLatLon
// 	}
// 	fm.M.RUnlock()

// 	peerCGs := fm.GetPeerPolledCachegroups()

// 	unpolledCGs := map[tc.CacheGroupName]struct{}{}
// 	for cg, _ := range cgs {
// 		unpolledCGs[cg] = struct{}{}
// 	}
// 	for _, cgs := range peerCGs {
// 		for cg, _ := range cgs {
// 			delete(unpolledCGs, cg)
// 		}
// 	}

// 	cgsToPoll := unpolledCGs // grab all unpolled CGs to poll

// 	type CGPolledNearby struct {
// 		PolledNear bool
// 		PolledFar  bool
// 	}

// }

func getDistanceFromLatLonInKm(lat1, lon1, lat2, lon2 float64) float64 {
	// Haversine formula, from https://www.movable-type.co.uk/scripts/latlong.html - MIT licensed
	R := 6371.0                  // Radius of the earth in km
	dLat := deg2rad(lat2 - lat1) // deg2rad below
	dLon := deg2rad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(deg2rad(lat1))*math.Cos(deg2rad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c // Distance in km
	return d
}

func deg2rad(deg float64) float64 { return deg * (math.Pi / 180) }

// func (fm *FakeMonitor) GetPeerPolledCachegroups() map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{} {

// }

//const MonitorPeerPolledCachegroupsUseFQDN = false // if false, use IP
// func (fm *FakeMonitor) GetPeerPolledCachegroups() map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{} {
// 	peerCGs := map[tc.TrafficMonitorName]map[tc.CacheGroupName]struct{}{}

// 	peers := map[tc.TrafficMonitorName]tc.CRConfigMonitor{}
// 	fm.M.RLock()
// 	for peerName, peerServer := range fm.CRConfig.Monitors {
// 		peers[tc.TrafficMonitorName(peerName)] = peerServer
// 	}
// 	fm.M.RUnlock()

// 	for peerName, peerServer := range peers {
// 		peerHost := "" // FQDN or IP to poll
// 		if MonitorPeerPolledCachegroupsUseFQDN {
// 			if peerServer.FQDN == nil {
// 				fmt.Println("ERROR: getting peer polled cachegroups: peer '" + peerName + "' has nil FQDN, skipping!")
// 				continue
// 			}
// 			peerHost = *peerServer.FQDN
// 		} else {
// 			if peerServer.IP == nil {
// 				fmt.Println("ERROR: getting peer polled cachegroups: peer '" + peerName + "' has nil IP, skipping!")
// 				continue
// 			}
// 			peerHost = *peerServer.IP
// 		}

// 		peerPort := 80
// 		if peerServer.Port != nil {
// 			peerPort = *peerServer.Port
// 		}

// 		peerURL := "http://" + peerHost + ":" + strconv.Itoa(peerPort) + PolledCacheGroupsPath

// 		resp, err := http.Get(peerURL)
// 		if err != nil {
// 			fmt.Println("ERROR: getting peer polled cachegroups: getting polled cachegroups from peer '" + string(peerName) + "': get error: " + err.Error())
// 			continue
// 		}
// 		defer resp.Body.Close()
// 		body, err := ioutil.ReadAll(resp.Body)
// 		if err != nil {
// 			fmt.Println("ERROR: getting peer polled cachegroups: getting polled cachegroups from peer '" + string(peerName) + "': reading body: " + err.Error())
// 			continue
// 		}

// 		peerCGList := []string{}
// 		if err := json.Unmarshal(body, &peerCGList); err != nil {
// 			fmt.Println("ERROR: getting peer polled cachegroups: getting polled cachegroups from peer '" + string(peerName) + "': decoding body: " + err.Error())
// 			continue
// 		}

// 		peerCGs[peerName] = map[tc.CacheGroupName]struct{}{}
// 		for _, cg := range peerCGs {
// 			peerCGs[peerName][cg] = struct{}{}
// 		}
// 	}
// 	return peerCGs
// }

/*

# TODO
- implement monitors seleting which other monitors to poll, for cache data of caches this monitor didn't poll.
  - include endpoint, to request and find out which peers were chosen

peer polling algorithm:

Data:
- all data of which monitors are polling which cachegroups is in TO
  - many-to-many table with 2 columns: monitor|polledCG
- MinStealCountThreshold is a Monitor data field (Parameter). See rules.
  - Suggest: 3

Rules:
- Monitors only assign CGs to themselves
  - never "assign" to someone else, because you don't know if they're alive, and can't "dictate work" to them. Only take upon yourself
- Monitors never unassign CGs from other running monitors.
  - Because you don't know that you're right and they're wrong. Monitors never presume rightness to the extent of forcibly changing another monitor.

- Monitors classify CGs as "near" and "far" based on distance to themselves (suggest: near = 1/4 the distance of the US)

- Two monitors are "equidistant" to a CG if they are both near or both far.

- If a CG is not polled by a equidistant  monitor, this monitor starts polling it.

- If a CG is unpolled by this monitor, periodically poll a monitor which is polling it, and aggregate its data (cache availability, stats, etc).
  - Because we want to poll the same monitor for multiple CGs in a single request, choosing which monitors to poll for which CGs is Knapsack.
    - Therefore, use a heuristic for Knapsack/Travelling Salesman. Most number of CGs assigned first is the obvious heuristic.

- If this monitor polls another monitor, and that monitor doesn't respond, or its response lacks the required CG, try the next monitor with that CG.

- If polling all monitors "assigned" a CG not assigned to this monitor fail, this monitor self-assigns that CG
  - assuming all monitors with it are dead.

- If a CG is assigned to an equidistant monitor with  >MinStealCountThreshold more caches than this monitor, this monitor self-assigns that CG
  - note threshold is caches, not cachegroups
  - threshold prevents flapping

- If a CG assigned to this monitor is also assigned to an equidistant monitor, and that monitor has >MinStealCountThreshold fewer caches than this monitor, this monitor self-unassigns that CG.
  - This should only be done after polling that monitor, to make sure it's working. Don't remove CGs without _verifying_ another monitor is polling first.

- If any monitor sees an OFFLINE or ADMIN_DOWN monitor assigned to a CG, in TO, it unassigns that monitor.
  - this violates the meta-rule that "monitors don't unassign other monitors," but monitors aren't trusted to shut themselves down safely. Therefore, other monitors "clean up" after offlined monitors. More specifically, "monitors don't unassign other running monitors."
  - Ideally this POST would use a If-Not-Modified, to prevent mutating data after something changed underneath us. But TO doesn't support that today.
    - Suggest: requiring If-Not-Modified support in new TO MonitorAssignedCacheGroups endpoint.

*/
