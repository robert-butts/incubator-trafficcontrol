package cache

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

// stats_type_astats is the default Stats format for Traffic Control.
// It is the Stats format produced by the `astats` plugin to Apache Traffic Server, included with Traffic Control.
//
// Stats are of the form `{"ats": {"name", number}}`,
// Where `name` is of the form:
//   `"plugin.remap_stats.fully-qualfiied-domain-name.example.net.stat-name"`
// Where `stat-name` is one of:
//   `in_bytes`, `out_bytes`, `status_2xx`, `status_3xx`, `status_4xx`, `status_5xx`

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/apache/incubator-trafficcontrol/lib/go-log"
	"github.com/apache/incubator-trafficcontrol/lib/go-tc"
	"github.com/apache/incubator-trafficcontrol/traffic_monitor/dsdata"
	"github.com/apache/incubator-trafficcontrol/traffic_monitor/todata"
)

func init() {
	AddStatsType("astats", astatsParse, astatsPrecompute)
}

func astatsParse(cache tc.CacheName, r io.Reader) (error, map[string]interface{}, AstatsSystem) {
	astats := Astats{}
	err := json.NewDecoder(r).Decode(&astats)
	return err, astats.Ats, astats.System
}

const KbPerMb = 1000

func astatsPrecompute(cache tc.CacheName, toData todata.TOData, rawStats map[string]interface{}, system AstatsSystem) PrecomputedData {
	stats := map[tc.DeliveryServiceName]dsdata.CacheDSStats{}
	precomputed := PrecomputedData{}
	err := error(nil)
	if precomputed.OutBytes, err = astatsOutBytes(system.ProcNetDev, system.InfName); err != nil {
		precomputed.OutBytes = 0
		err := errors.New("precomputing outbytes: " + err.Error())
		log.Errorln("astatsPrecompute '" + string(cache) + "' handle: " + err.Error())
		precomputed.Errors = append(precomputed.Errors, err)
	}
	precomputed.MaxKbps = int64(system.InfSpeed) * KbPerMb

	for stat, ival := range rawStats {
		fval, ok := ival.(float64)
		if !ok {
			err := fmt.Errorf("stat '%s' value unknown type %T", stat, ival)
			log.Infof("precomputing cache %v stat %v value %v error %v", cache, stat, ival, err)
			precomputed.Errors = append(precomputed.Errors, err)
			continue
		}
		val := int64(fval)

		err := error(nil)
		stats, err = astatsProcessStat(cache, stats, toData, stat, val)
		if err != nil && err != dsdata.ErrNotProcessedStat {
			log.Infof("precomputing cache %v stat %v value %v error %v", cache, stat, val, err)
			precomputed.Errors = append(precomputed.Errors, err)
		}
	}
	precomputed.DSStats = stats
	return precomputed
}

// outBytes takes the proc.net.dev string, and the interface name, and returns the bytes field
func astatsOutBytes(procNetDev, iface string) (int64, error) {
	if procNetDev == "" {
		return 0, errors.New("procNetDev empty")
	}
	if iface == "" {
		return 0, errors.New("iface empty")
	}
	ifacePos := strings.Index(procNetDev, iface)
	if ifacePos == -1 {
		return 0, errors.New("interface '" + iface + "' not found in proc.net.dev '" + procNetDev + "'")
	}

	procNetDevIfaceBytes := procNetDev[ifacePos+len(iface)+1:]
	procNetDevIfaceBytesArr := strings.Fields(procNetDevIfaceBytes) // TODO test
	if len(procNetDevIfaceBytesArr) < 10 {
		return 0, errors.New("proc.net.dev iface '" + iface + "' unknown format '" + procNetDev + "'")
	}
	procNetDevIfaceBytes = procNetDevIfaceBytesArr[8]

	return strconv.ParseInt(procNetDevIfaceBytes, 10, 64)
}

func astatsProcessStat(server tc.CacheName, stats map[tc.DeliveryServiceName]dsdata.CacheDSStats, toData todata.TOData, stat string, val int64) (map[tc.DeliveryServiceName]dsdata.CacheDSStats, error) {
	dsStatPrefix := "plugin.remap_stats."
	if !strings.HasPrefix(stat, dsStatPrefix) {
		return stats, dsdata.ErrNotProcessedStat
	}
	stat = stat[len(dsStatPrefix):]
	lastDot := strings.LastIndex(stat, ".")
	if lastDot == -1 {
		return stats, errors.New("malformed stat '" + stat + "'")
	}
	fqdn := stat[:lastDot]
	statName := stat[lastDot+1:]
	ds, ok := toData.DeliveryServiceRegexes.FQDNDeliveryService(fqdn)
	if !ok {
		return stats, errors.New("no delivery service match for fqdn '" + fqdn + "' stat '" + statName + "'")
	}
	dsStat := stats[ds]
	if err := astatsAddCacheStat(&dsStat, statName, val); err != nil {
		return stats, err
	}
	stats[ds] = dsStat
	return stats, nil
}

// addCacheStat adds the given stat to the existing stat.
func astatsAddCacheStat(stat *dsdata.CacheDSStats, name string, val int64) error {
	switch name {
	case "status_2xx":
		stat.Status2xx += val
	case "status_3xx":
		stat.Status3xx += val
	case "status_4xx":
		stat.Status4xx += val
	case "status_5xx":
		stat.Status5xx += val
	case "out_bytes":
		stat.OutBytes += val
	case "in_bytes":
		stat.InBytes += val
	case "status_unknown":
		return dsdata.ErrNotProcessedStat
	default:
		return errors.New("unknown stat '" + name + "'") // TODO verify removed stats aren't used, e.g. is_available
	}
	return nil
}
