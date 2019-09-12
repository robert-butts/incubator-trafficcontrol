package v14

/*

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"testing"
)

func TestServerCapabilities(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, ServerCapabilityTypes, ServerCapabilities}, func() {
		GetTestServerCapabilities(t)
	})
}

func CreateTestServerCapabilities(t *testing.T) {
	edgeServer := testData.Servers[0]
	for _, server := range testData.Servers {
		if server.Type != "EDGE" {
			continue
		}
		edgeServer = server
		break
	}
	if edgeServer.Type != "EDGE" {
		t.Fatalf("no edge server in test data, must have at least 1 to test\n")
	}

	serverCapabilityTypes, _, err := TOSession.GetServerCapabilityTypes()
	if err != nil {
		t.Fatalf("getting server capability types: " + err.Error())
	}
	if len(serverCapabilityTypes) == 0 {
		t.Errorf("no server capability types in Traffic Ops, must have at least 1 to test")
	}

	_, _, err = TOSession.UpdateServerCapabilities(edgeServer.ID, []string{*serverCapabilityTypes[0].Name})
	if err != nil {
		t.Fatalf("adding server capability: " + err.Error())
	}

	serverCapability, _, err := TOSession.GetServerCapabilitiesByServerID(edgeServer.ID)
	if len(serverCapability.Capabilities) != 1 || serverCapability.Capabilities[0] != *serverCapabilityTypes[0].Name {
		t.Errorf("server capability expected: %v actual %v", serverCapabilityTypes[0].Name, serverCapability.Capabilities)
	}
}

func GetTestServerCapabilities(t *testing.T) {
	serverCapabilities, _, err := TOSession.GetServerCapabilities()
	if err != nil {
		t.Fatalf("getting GetDeliveryServiceRequiredCapabilities: " + err.Error())
	}
	if len(serverCapabilities) == 0 {
		t.Errorf("get returned no capabilities")
	}
}

func DeleteTestServerCapabilities(t *testing.T) {
	edgeServer := testData.Servers[0]
	for _, server := range testData.Servers {
		if server.Type != "EDGE" {
			continue
		}
		edgeServer = server
		break
	}
	if edgeServer.Type != "EDGE" {
		t.Fatalf("no edge server in test data, must have at least 1 to test\n")
	}

	serverCapability, _, err := TOSession.GetServerCapabilitiesByServerID(edgeServer.ID)
	if len(serverCapability.Capabilities) == 0 {
		t.Errorf("server '" + edgeServer.HostName + "' has no capabilities, must have at least 1 to test")
	}

	_, _, err = TOSession.UpdateServerCapabilities(edgeServer.ID, nil)
	if err != nil {
		t.Fatalf("deleting server capabilities: " + err.Error())
	}
}
