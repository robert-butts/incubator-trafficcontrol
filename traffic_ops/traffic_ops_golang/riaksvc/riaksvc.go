package riaksvc

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

// TODO remove this package, when TO has been upgraded to the next major version, and all deprecated uses are removed.

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"

	"github.com/basho/riak-go-client"
)

const DefaultHealthCheckInterval = time.Second * 5

var HealthCheckInterval time.Duration

func GetRiakConfig(riakConfigFile string) (*riak.AuthOptions, error) {
	riakConfString, err := ioutil.ReadFile(riakConfigFile)
	if err != nil {
		return nil, fmt.Errorf("reading riak conf '%v': %v", riakConfigFile, err)
	}

	riakConfBytes := []byte(riakConfString)

	rconf := &riak.AuthOptions{}
	rconf.TlsConfig = &tls.Config{}
	err = json.Unmarshal(riakConfBytes, &rconf)
	if err != nil {
		return nil, fmt.Errorf("Unmarshaling riak conf '%v': %v", riakConfigFile, err)
	}

	type config struct {
		Hci string `json:"HealthCheckInterval"`
	}

	var checkconfig config
	err = json.Unmarshal(riakConfBytes, &checkconfig)
	if err == nil {
		hci, _ := time.ParseDuration(checkconfig.Hci)
		if 0 < hci {
			HealthCheckInterval = hci
		}
	} else {
		log.Infoln("Error unmarshalling riak config options: " + err.Error())
	}

	if HealthCheckInterval <= 0 {
		HealthCheckInterval = DefaultHealthCheckInterval
		log.Infoln("HeathCheckInterval override")
	}

	log.Infoln("Riak health check interval set to:", HealthCheckInterval)

	return rconf, nil
}
