package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

// PeerPoller is an object for managing polling Monitor peers.
//
// It has functions which take a new set of peers to poll,
// which will automatically figure out the difference in what's already being polled,
// and stop the appropriate pollers, and create the new ones.
//
type MonitorPeerPollManager struct {
	PollInterval time.Duration
	Pollers      map[tc.TrafficMonitorName]*MonitorPeerPoller
	Results      chan MonitorPeerPollResult
}

// Stop stops the manager. Once stopped, it cannot be used again. Create a new one.
// Stop closes the Results chan. To stop selecting or ranging over it,
// readers should check for the channel to become closed.
func (pm *MonitorPeerPollManager) Stop() {
	for _, pl := range pm.Pollers {
		pl.Die <- struct{}{}
	}
	close(pm.Results)
}

type MonitorPeerPollResultFuncMonitorPeerPollResultFunc func(MonitorPeerPollResult)

type MonitorPeerPollResultDataT []tc.CacheGroupName
type MonitorPeerPollResult struct {
	MonitorName tc.TrafficMonitorName
	Err         error
	Data        MonitorPeerPollResultDataT
}

const MonitorPeerPollerPath = `/api/polled-cachegroups` // TODO config?
const MonitorPeerPollResultChanBufferSize = 10          // TODO config?
const MonitorPeerPollTimeout = time.Second * 3          // TODO config?
const MonitorPeerPollerUseHostName = false              // TODO config?

func NewMonitorPeerPollManager(pollInterval time.Duration) *MonitorPeerPollManager {
	return &MonitorPeerPollManager{
		PollInterval: pollInterval,
		Pollers:      map[tc.TrafficMonitorName]*MonitorPeerPoller{},
		Results:      make(chan MonitorPeerPollResult, MonitorPeerPollResultChanBufferSize),
	}
}

func NewMonitorPeerPoller(
	interval time.Duration,
	result chan MonitorPeerPollResult,
	monitor tc.TrafficMonitor,
) *MonitorPeerPoller {
	return &MonitorPeerPoller{
		Interval: interval,
		Result:   result,
		Die:      make(chan struct{}),
		Monitor:  monitor,
	}
}

// MonitorPeerPoller is immutable after creation, for thread-safety.
// Chans can be written to, but other data may not be modified.
type MonitorPeerPoller struct {
	Interval time.Duration
	Result   chan MonitorPeerPollResult
	Die      chan struct{}

	Monitor tc.TrafficMonitor
	CG      map[tc.CacheGroupName]struct{}
}

func (po *MonitorPeerPoller) Start() {
	go func() {
		timer := time.NewTimer(po.Interval)
		for {
			select {
			case <-po.Die:
				timer.Stop()
				return
			case <-timer.C:
				start := time.Now()
				po.Poll()
				pollTime := time.Since(start)
				timer.Reset(po.Interval - pollTime)
			}
		}
	}()
}

func (po *MonitorPeerPoller) Poll() {
	reqURI := po.PollURI()
	httpClient := po.HTTPClient()
	resp, err := httpClient.Get(reqURI)
	if err != nil {
		po.Result <- MonitorPeerPollResult{MonitorName: tc.TrafficMonitorName(po.Monitor.HostName), Err: errors.New("requesting: " + err.Error())}
		return
	}
	defer resp.Body.Close()

	data := MonitorPeerPollResultDataT{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		po.Result <- MonitorPeerPollResult{MonitorName: tc.TrafficMonitorName(po.Monitor.HostName), Err: errors.New("decoding: " + err.Error())}
		return
	}
	po.Result <- MonitorPeerPollResult{MonitorName: tc.TrafficMonitorName(po.Monitor.HostName), Data: data}
}

func (po *MonitorPeerPoller) PollURI() string {
	scheme := "http"
	schemeDefaultPort := 80
	reqURI := scheme + "://"
	if MonitorPeerPollerUseHostName {
		reqURI += po.Monitor.HostName
	} else {
		reqURI += po.Monitor.IP
	}
	if po.Monitor.Port != 0 && po.Monitor.Port != schemeDefaultPort {
		reqURI += ":" + strconv.Itoa(po.Monitor.Port)
	}
	reqURI += MonitorPeerPollerPath
	return reqURI
}

func (po *MonitorPeerPoller) HTTPClient() *http.Client {
	// TODO reuse, for performance?
	return &http.Client{
		Timeout: MonitorPeerPollTimeout,
	}
}

type MonitorPeerPollData struct {
	Monitor tc.TrafficMonitor
}

func (pm *MonitorPeerPollManager) RefreshPollers(newPolls []MonitorPeerPollData, dbgOwner tc.TrafficMonitorName) {
	fmt.Printf("DEBUG MonitorPeerPollManager.RefreshPollers %v newPolls %v\n", dbgOwner, len(newPolls))
	newMonitorNames := map[tc.TrafficMonitorName]struct{}{}
	for _, newData := range newPolls {
		newMonitorNames[tc.TrafficMonitorName(newData.Monitor.HostName)] = struct{}{}
	}

	// first, kill old pollers not in the new list.
	for monitorName, poller := range pm.Pollers {
		if _, ok := newMonitorNames[monitorName]; ok {
			fmt.Printf("DEBUG MonitorPeerPollManager.RefreshPollers %v newMonitor %v exists, continuing\n", dbgOwner, monitorName)
			continue
		}
		fmt.Printf("DEBUG MonitorPeerPollManager.RefreshPollers %v existing %v not in new, killing\n", dbgOwner, monitorName)
		poller.Die <- struct{}{}
		delete(pm.Pollers, monitorName)
	}

	// Now, add new pollers, and refresh CGs on existing ones.
	for _, newPoll := range newPolls {
		if _, ok := pm.Pollers[tc.TrafficMonitorName(newPoll.Monitor.HostName)]; ok {
			continue
		}
		fmt.Printf("DEBUG MonitorPeerPollManager.RefreshPollers %v newing %v\n", dbgOwner, tc.TrafficMonitorName(newPoll.Monitor.HostName))
		// TODO sleep rand%interval before starting, to offset polls?
		poller := NewMonitorPeerPoller(pm.PollInterval, pm.Results, newPoll.Monitor)
		poller.Start()
		pm.Pollers[tc.TrafficMonitorName(newPoll.Monitor.HostName)] = poller
	}
}
