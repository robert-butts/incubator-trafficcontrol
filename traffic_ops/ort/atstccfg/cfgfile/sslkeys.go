package cfgfile

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
	"encoding/base64"

	"github.com/apache/trafficcontrol/lib/go-atscfg"
	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
)

func GetSSLCertsAndKeyFiles(toData *TOData) ([]ATSConfigFile, error) {
	dses := atscfg.DeliveryServicesToSSLMultiCertDSes(toData.DeliveryServices)
	dses = atscfg.GetSSLMultiCertDotConfigDeliveryServices(dses)

	configs := []ATSConfigFile{}
	for _, keys := range toData.SSLKeys {
		dsName := tc.DeliveryServiceName(keys.DeliveryService)
		ds, ok := dses[dsName]
		if !ok {
			continue
		}

		cert, err := base64.StdEncoding.DecodeString(keys.Certificate.Crt)
		if err != nil {
			log.Errorln("Delivery Service '" + string(dsName) + "' skipping HTTPS certificate! Failed to decode cert: " + err.Error())
			continue
		}
		if len(cert) > 0 && cert[len(cert)-1] != '\n' {
			cert = append(cert, '\n') // it's going to be a file, needs a trailing newline to be POSIX-compliant.
		}

		key, err := base64.StdEncoding.DecodeString(keys.Certificate.Key)
		if err != nil {
			log.Errorln("Delivery Service '" + string(dsName) + "' skipping HTTPS certificate! Failed to decode key: " + err.Error())
			continue
		}
		if len(key) > 0 && key[len(key)-1] != '\n' {
			key = append(key, '\n') // it's going to be a file, needs a trailing newline to be POSIX-compliant.
		}

		certName, keyName := atscfg.GetSSLMultiCertDotConfigCertAndKeyName(dsName, ds)

		keyFile := ATSConfigFile{}
		keyFile.FileNameOnDisk = keyName
		keyFile.Location = "/opt/trafficserver/etc/trafficserver/ssl/" // TODO read config, don't hard code
		keyFile.Text = string(key)
		configs = append(configs, keyFile)

		certFile := ATSConfigFile{}
		certFile.FileNameOnDisk = certName
		certFile.Location = "/opt/trafficserver/etc/trafficserver/ssl/" // TODO read config, don't hard code
		certFile.Text = string(cert)
		configs = append(configs, certFile)
	}

	return configs, nil
}
