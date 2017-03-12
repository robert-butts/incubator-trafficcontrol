// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config_files

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	towrap "github.com/apache/incubator-trafficcontrol/traffic_monitor_golang/traffic_monitor/trafficopswrapper"
	to "github.com/apache/incubator-trafficcontrol/traffic_ops/client"
	"io/ioutil"
	"net/http"
)

const URLSigKeysBucket = "url_sig_keys"

func getRiakURL(toClient towrap.ITrafficOpsSession) (string, error) {
	servers, err := toClient.Servers()
	if err != nil {
		return "", fmt.Errorf("getting servers: %v", err)
	}

	riakServer := to.Server{}
	for _, server := range servers {
		if server.Type == "RIAK" {
			riakServer = server
			break
		}
	}
	if riakServer == (to.Server{}) {
		return "", fmt.Errorf("no riak server found")
	}

	if riakServer.TCPPort != 0 {
		return fmt.Sprintf("%s.%s:%s", riakServer.HostName, riakServer.DomainName, riakServer.TCPPort), nil
	}
	return fmt.Sprintf("%s.%s", riakServer.HostName, riakServer.DomainName), nil

}

// RiakGetUrlSigKeys takes a URL sig filename (which should be 'url_sig_{deliveryservice}.config'), and returns the map of key names to values.
func RiakGetURLSigKeys(toClient towrap.ITrafficOpsSession, user, pass, file string, insecure bool) (map[string]string, error) {
	riakFQDN, err := getRiakURL(toClient)
	if err != nil {
		return nil, fmt.Errorf("getting Riak URL: %v", err)
	}

	url := fmt.Sprintf("https://%s:%s@%s/riak/%s/%s", user, pass, riakFQDN, URLSigKeysBucket, file)

	httpClient := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}}
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("getting Riak keys: %v", err)
	}
	defer resp.Body.Close()

	keys := map[string]string{}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %v", err)
	}

	err = json.Unmarshal(bytes, &keys)
	// err = json.NewDecoder(resp.Body).Decode(keys)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling JSON url '%v' bytes '%v': %v", url, string(bytes), err)
	}
	return keys, nil
}
