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

import (
	"bytes"
	"errors"
	"strconv"
)

// Astats contains ATS data returned from the Astats ATS plugin. This includes generic stats, as well as fixed system stats.
type Astats struct {
	Ats    map[string]interface{} `json:"ats"`
	System AstatsSystem           `json:"system"`
}

// AstatsSystem represents fixed system stats returne from ATS by the Astats plugin.
type AstatsSystem struct {
	InfName           string                 `json:"inf.name"`
	InfSpeed          int                    `json:"inf.speed"`
	ProcNetDev        AstatsSystemProcNetDev `json:"proc.net.dev"`
	ProcLoadavg       AstatsSystemLoadAvg    `json:"proc.loadavg"`
	ConfigLoadRequest int                    `json:"configReloadRequests"`
	LastReloadRequest int                    `json:"lastReloadRequest"`
	ConfigReloads     int                    `json:"configReloads"`
	LastReload        int                    `json:"lastReload"`
	AstatsLoad        int                    `json:"astatsLoad"`
	NotAvailable      bool                   `json:"notAvailable,omitempty"`
}

type AstatsSystemLoadAvg struct {
	CPU1m        float64
	CPU5m        float64
	CPU10m       float64
	RunningProcs uint64
	TotalProcs   uint64
	LastPIDUsed  uint64
}

// UnmarshalJSON unmarshals a JSON string of the form of Linux' /proc/loadavg
func (a *AstatsSystemLoadAvg) UnmarshalJSON(data []byte) error {
	if len(data) < 2 {
		return errors.New("malformed load average string, not a JSON string: '" + string(data) + "'")
	}
	data = data[1 : len(data)-1] // strip quotes
	fields := bytes.Fields(data)

	if len(fields) < 5 {
		return errors.New("malformed load average string, not enough fields: '" + string(data) + "'")
	}
	var err error
	if a.CPU1m, err = strconv.ParseFloat(string(fields[0]), 64); err != nil {
		return errors.New("malformed load average string, CPU1m not a float: '" + string(data) + "'")
	}
	if a.CPU5m, err = strconv.ParseFloat(string(fields[1]), 64); err != nil {
		return errors.New("malformed load average string, CPU5m not a float: '" + string(data) + "'")
	}
	if a.CPU10m, err = strconv.ParseFloat(string(fields[2]), 64); err != nil {
		return errors.New("malformed load average string, CPU10m not a float: '" + string(data) + "'")
	}

	procsRunAndTotal := bytes.Split(fields[3], []byte("/"))
	if len(procsRunAndTotal) < 2 {
		return errors.New("malformed load average string, Running/Total procs malformed: '" + string(data) + "'")
	}
	if a.RunningProcs, err = strconv.ParseUint(string(procsRunAndTotal[0]), 10, 64); err != nil {
		return errors.New("malformed load average string, running procs not a unsigned int: '" + string(data) + "'")
	}
	if a.TotalProcs, err = strconv.ParseUint(string(procsRunAndTotal[1]), 10, 64); err != nil {
		return errors.New("malformed load average string, total procs not an unsigned int: '" + string(data) + "'")
	}

	if a.LastPIDUsed, err = strconv.ParseUint(string(fields[4]), 10, 64); err != nil {
		return errors.New("malformed load average string, last PID not an unsigned int: '" + string(data) + "'")
	}
	return nil
}

type AstatsSystemProcNetDev struct {
	Interface      string
	RcvBytes       uint64
	RcvPackets     uint64
	RcvErrs        uint64
	RcvDropped     uint64
	RcvFIFOErrs    uint64
	RcvFrameErrs   uint64
	RcvCompressed  uint64
	RcvMulticast   uint64
	SndBytes       uint64
	SndPackets     uint64
	SndErrs        uint64
	SndDropped     uint64
	SndFIFOErrs    uint64
	SndCollisions  uint64
	SndCarrierErrs uint64
	SndCompressed  uint64
}

// UnmarshalJSON unmarshals a JSON string of the form of Linux' /proc/net/dev
func (a *AstatsSystemProcNetDev) UnmarshalJSON(data []byte) error {
	if len(data) < 2 {
		return errors.New("malformed proc net dev string, not a JSON string: '" + string(data) + "'")
	}
	data = data[1 : len(data)-1] // strip quotes
	fields := bytes.Fields(data)
	// TODO fix for interfacename:firstfield with no spacing
	if len(fields) < 17 {
		return errors.New("malformed proc net dev string, not enough fields: '" + string(data) + "'")
	}

	if len(fields[0]) < 2 {
		return errors.New("malformed proc net dev string, missing interface: '" + string(data) + "'")
	}

	a.Interface = string(fields[0][:len(fields[0])-1])

	var err error
	if a.RcvBytes, err = strconv.ParseUint(string(fields[1]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, RcvBytes not a uint: '" + string(data) + "'")
	}
	if a.RcvPackets, err = strconv.ParseUint(string(fields[2]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, RcvPackets not a uint: '" + string(data) + "'")
	}
	if a.RcvErrs, err = strconv.ParseUint(string(fields[3]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, RcvErrs not a uint: '" + string(data) + "'")
	}
	if a.RcvDropped, err = strconv.ParseUint(string(fields[4]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, RcvDropped not a uint: '" + string(data) + "'")
	}
	if a.RcvFIFOErrs, err = strconv.ParseUint(string(fields[5]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, RcvFIFOErrs not a uint: '" + string(data) + "'")
	}
	if a.RcvFrameErrs, err = strconv.ParseUint(string(fields[6]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, RcvFrameErrs not a uint: '" + string(data) + "'")
	}
	if a.RcvCompressed, err = strconv.ParseUint(string(fields[7]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, RcvCompressed not a uint: '" + string(data) + "'")
	}
	if a.RcvMulticast, err = strconv.ParseUint(string(fields[8]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, RcvMulticast not a uint: '" + string(data) + "'")
	}
	if a.SndBytes, err = strconv.ParseUint(string(fields[9]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, SndBytes not a uint: '" + string(data) + "'")
	}
	if a.SndPackets, err = strconv.ParseUint(string(fields[10]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, SndPackets not a uint: '" + string(data) + "'")
	}
	if a.SndErrs, err = strconv.ParseUint(string(fields[11]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, SndErrs not a uint: '" + string(data) + "'")
	}
	if a.SndDropped, err = strconv.ParseUint(string(fields[12]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, SndDropped not a uint: '" + string(data) + "'")
	}
	if a.SndFIFOErrs, err = strconv.ParseUint(string(fields[13]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, SndFIFOErrs not a uint: '" + string(data) + "'")
	}
	if a.SndCollisions, err = strconv.ParseUint(string(fields[14]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, SndCollisions not a uint: '" + string(data) + "'")
	}
	if a.SndCarrierErrs, err = strconv.ParseUint(string(fields[15]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, SndCarrierErrs not a uint: '" + string(data) + "'")
	}
	if a.SndCompressed, err = strconv.ParseUint(string(fields[16]), 10, 64); err != nil {
		return errors.New("malformed proc net dev string, SndCompressed not a uint: '" + string(data) + "'")
	}
	return nil
}

// MarshalJSON marshals into a JSON string of the form of Linux' /proc/loadavg
func (a AstatsSystemLoadAvg) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`"`)
	buffer.WriteString(strconv.FormatFloat(a.CPU1m, 'f', -1, 64))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatFloat(a.CPU5m, 'f', -1, 64))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatFloat(a.CPU10m, 'f', -1, 64))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.RunningProcs, 10))
	buffer.WriteString("/")
	buffer.WriteString(strconv.FormatUint(a.TotalProcs, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.LastPIDUsed, 10))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// MarshalJSON marshals into a JSON string of the form of Linux' /proc/net/dev
func (a AstatsSystemProcNetDev) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`"`)
	buffer.WriteString(a.Interface)
	buffer.WriteString(": ")
	buffer.WriteString(strconv.FormatUint(a.RcvBytes, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.RcvPackets, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.RcvErrs, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.RcvDropped, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.RcvFIFOErrs, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.RcvFrameErrs, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.RcvCompressed, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.RcvMulticast, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.SndBytes, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.SndPackets, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.SndErrs, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.SndDropped, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.SndFIFOErrs, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.SndCollisions, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.SndCarrierErrs, 10))
	buffer.WriteString(" ")
	buffer.WriteString(strconv.FormatUint(a.SndCompressed, 10))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}
