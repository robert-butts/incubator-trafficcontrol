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
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestAstats(t *testing.T) {
	t.Log("Running Astats Tests")

	text, err := ioutil.ReadFile("astats.json")
	if err != nil {
		t.Fatalf(`reading Astats example file expected success, actual %+v`, err)
	}
	astats := Astats{}
	err = json.Unmarshal(text, &astats)
	if err != nil {
		t.Fatalf(`unmarshalling Astats expected success, actual %+v`, err)
	}
}

func TestAstatsSystemLoadAvgUnmarshalJSON(t *testing.T) {
	obj := struct {
		LoadAvg AstatsSystemLoadAvg `json: "loadavg"`
	}{}
	j := `{"loadavg": "0.20 0.07 0.08 1/967 29536"}`

	err := json.Unmarshal([]byte(j), &obj)
	if err != nil {
		t.Fatalf(`unmarshalling Astats expected success, actual %+v`, err)
	}
	if obj.LoadAvg.CPU1m != 0.20 {
		t.Fatalf(`AstatsSystemLoadAvg.UnmarshalJSON expected CPU1m 0.20, actual %+v`, obj.LoadAvg.CPU1m)
	}
	if obj.LoadAvg.CPU5m != 0.07 {
		t.Fatalf(`AstatsSystemLoadAvg.UnmarshalJSON expected CPU5m 0.07, actual %+v`, obj.LoadAvg.CPU5m)
	}
	if obj.LoadAvg.CPU10m != 0.08 {
		t.Fatalf(`AstatsSystemLoadAvg.UnmarshalJSON expected CPU1m 0.08, actual %+v`, obj.LoadAvg.CPU10m)
	}
	if obj.LoadAvg.RunningProcs != 1 {
		t.Fatalf(`AstatsSystemLoadAvg.UnmarshalJSON expected RunningProcs 1, actual %+v`, obj.LoadAvg.RunningProcs)
	}
	if obj.LoadAvg.TotalProcs != 967 {
		t.Fatalf(`AstatsSystemLoadAvg.UnmarshalJSON expected TotalProcs 967, actual %+v`, obj.LoadAvg.TotalProcs)
	}
	if obj.LoadAvg.LastPIDUsed != 29536 {
		t.Fatalf(`AstatsSystemLoadAvg.UnmarshalJSON expected LastPIDUsed 29535, actual %+v`, obj.LoadAvg.LastPIDUsed)
	}
}

func TestAstatsSystemProcNetDevUnmarshalJSON(t *testing.T) {
	obj := struct {
		ProcNetDev AstatsSystemProcNetDev `json:"procnetdev"`
	}{}
	j := `{"procnetdev": "bond0: 42 24    1    2    3     4          5   123 456 789    6    7    8     9       10          11"}`

	err := json.Unmarshal([]byte(j), &obj)
	if err != nil {
		t.Fatalf(`unmarshalling Astats expected success, actual %+v`, err)
	}

	if obj.ProcNetDev.Interface != "bond0" {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected Interface 'bond0', actual '%+v'`, obj.ProcNetDev.Interface)
	}
	if obj.ProcNetDev.RcvBytes != 42 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected RcvBytes 42, actual %+v`, obj.ProcNetDev.RcvBytes)
	}
	if obj.ProcNetDev.RcvPackets != 24 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected RcvPackets 24, actual %+v`, obj.ProcNetDev.RcvPackets)
	}
	if obj.ProcNetDev.RcvErrs != 1 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected RcvErrs 1, actual %+v`, obj.ProcNetDev.RcvErrs)
	}
	if obj.ProcNetDev.RcvDropped != 2 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected RcvDropped 2, actual %+v`, obj.ProcNetDev.RcvDropped)
	}
	if obj.ProcNetDev.RcvFIFOErrs != 3 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected RcvFIFOErrs 3, actual %+v`, obj.ProcNetDev.RcvFIFOErrs)
	}
	if obj.ProcNetDev.RcvFrameErrs != 4 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected RcvFrameErrs 4, actual %+v`, obj.ProcNetDev.RcvFrameErrs)
	}
	if obj.ProcNetDev.RcvCompressed != 5 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected RcvCompressed 4, actual %+v`, obj.ProcNetDev.RcvCompressed)
	}
	if obj.ProcNetDev.RcvMulticast != 123 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected RcvMulticast 123, actual %+v`, obj.ProcNetDev.RcvMulticast)
	}
	if obj.ProcNetDev.SndBytes != 456 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected SndBytes 456, actual %+v`, obj.ProcNetDev.SndBytes)
	}
	if obj.ProcNetDev.SndPackets != 789 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected SndPackets 789, actual %+v`, obj.ProcNetDev.SndPackets)
	}
	if obj.ProcNetDev.SndErrs != 6 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected SndErrs 6, actual %+v`, obj.ProcNetDev.SndErrs)
	}
	if obj.ProcNetDev.SndDropped != 7 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected SndDropped 7, actual %+v`, obj.ProcNetDev.SndDropped)
	}
	if obj.ProcNetDev.SndFIFOErrs != 8 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected SndFIFOErrs 8, actual %+v`, obj.ProcNetDev.SndFIFOErrs)
	}
	if obj.ProcNetDev.SndCollisions != 9 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected SndCollisions 9, actual %+v`, obj.ProcNetDev.SndCollisions)
	}
	if obj.ProcNetDev.SndCarrierErrs != 10 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected SndCarrierErrs 10, actual %+v`, obj.ProcNetDev.SndCarrierErrs)
	}
	if obj.ProcNetDev.SndCompressed != 11 {
		t.Fatalf(`AstatsSystemProcNetDev.UnmarshalJSON expected SndCompressed 11, actual %+v`, obj.ProcNetDev.SndCompressed)
	}
}

func TestAstatsSystemLoadAvgMarshalJSON(t *testing.T) {
	l := AstatsSystemLoadAvg{
		CPU1m:        1.2,
		CPU5m:        7.4,
		CPU10m:       4.5,
		RunningProcs: 1,
		TotalProcs:   2,
		LastPIDUsed:  8,
	}
	obj := struct {
		LoadAvg AstatsSystemLoadAvg `json:"loadavg"`
	}{LoadAvg: l}

	bytes, err := json.Marshal(obj)
	expected := `{"loadavg":"1.2 7.4 4.5 1/2 8"}`
	if err != nil {
		t.Fatalf(`AstatsSystemLoadAvg.MarshalJSON expected error nil, actual %+v`, err)
	}
	if string(bytes) != expected {
		t.Fatalf(`AstatsSystemLoadAvg.MarshalJSON expected %v, actual %+v`, expected, string(bytes))
	}
}

func TestAstatsSystemProcNetDevMarshalJSON(t *testing.T) {
	p := AstatsSystemProcNetDev{
		Interface:      "foo",
		RcvBytes:       1,
		RcvPackets:     2,
		RcvErrs:        3,
		RcvDropped:     4,
		RcvFIFOErrs:    5,
		RcvFrameErrs:   6,
		RcvCompressed:  7,
		RcvMulticast:   8,
		SndBytes:       9,
		SndPackets:     10,
		SndErrs:        11,
		SndDropped:     12,
		SndFIFOErrs:    13,
		SndCollisions:  14,
		SndCarrierErrs: 15,
		SndCompressed:  16,
	}

	obj := struct {
		ProcNetDev AstatsSystemProcNetDev `json:"procnetdev"`
	}{ProcNetDev: p}

	bytes, err := json.Marshal(obj)
	expected := `{"procnetdev":"foo: 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16"}`
	if err != nil {
		t.Fatalf(`AstatsSystemProcNetDev.MarshalJSON expected error nil, actual %+v`, err)
	}
	if string(bytes) != expected {
		t.Fatalf(`AstatsSystemProcNetDev.MarshalJSON expected %v, actual %+v`, expected, string(bytes))
	}
}
