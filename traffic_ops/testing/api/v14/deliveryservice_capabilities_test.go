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

func TestDeliveryServiceCapabilities(t *testing.T) {
	WithObjs(t, []TCObj{CDNs, Types, Parameters, Profiles, Statuses, Divisions, Regions, PhysLocations, CacheGroups, Servers, ServerCapabilityTypes, DeliveryServicesRequiredCapabilities}, func() {
		GetTestDeliveryServiceRequiredCapabilities(t)
	})
}

func CreateTestDeliveryServiceRequiredCapabilities(t *testing.T) {
	httpDS := testData.DeliveryServices[0]
	for _, ds := range testData.DeliveryServices {
		if ds.Type != "HTTP" {
			continue
		}
		httpDS = ds
		break
	}
	if httpDS.Type != "HTTP" {
		t.Fatalf("no HTTP DS in test data, must have at least 1 to test\n")
	}

	serverCapabilityTypes, _, err := TOSession.GetServerCapabilityTypes()
	if err != nil {
		t.Fatalf("getting server capability types: " + err.Error())
	}
	if len(serverCapabilityTypes) == 0 {
		t.Errorf("no server capability types in Traffic Ops, must have at least 1 to test")
	}

	_, _, err = TOSession.UpdateDeliveryServiceRequiredCapabilities(httpDS.ID, []string{*serverCapabilityTypes[0].Name})
	if err != nil {
		t.Fatalf("adding ds required capability: " + err.Error())
	}

	dsRequiredCapability, _, err := TOSession.GetDeliveryServiceRequiredCapabilitiesByDeliveryServiceID(httpDS.ID)
	if len(dsRequiredCapability.Capabilities) != 1 || dsRequiredCapability.Capabilities[0] != *serverCapabilityTypes[0].Name {
		t.Errorf("ds required capability expected: %v actual %v", serverCapabilityTypes[0].Name, dsRequiredCapability.Capabilities)
	}
}

func GetTestDeliveryServiceRequiredCapabilities(t *testing.T) {
	dsRequiredCapabilities, _, err := TOSession.GetDeliveryServiceRequiredCapabilities()
	if err != nil {
		t.Fatalf("getting GetDeliveryServiceRequiredCapabilities: " + err.Error())
	}
	if len(dsRequiredCapabilities) == 0 {
		t.Errorf("get returned no capabilities")
	}
}

func DeleteTestDeliveryServiceRequiredCapabilities(t *testing.T) {
	httpDS := testData.DeliveryServices[0]
	for _, ds := range testData.DeliveryServices {
		if ds.Type != "HTTP" {
			continue
		}
		httpDS = ds
		break
	}
	if httpDS.Type != "HTTP" {
		t.Fatalf("no HTTP DS in test data, must have at least 1 to test\n")
	}

	dsRequiredCapabilities, _, err := TOSession.GetDeliveryServiceRequiredCapabilitiesByDeliveryServiceID(httpDS.ID)
	if len(dsRequiredCapabilities.Capabilities) == 0 {
		t.Errorf("server '" + httpDS.XMLID + "' has no capabilities, must have at least 1 to test")
	}

	_, _, err = TOSession.UpdateDeliveryServiceRequiredCapabilities(httpDS.ID, nil)
	if err != nil {
		t.Fatalf("deleting delivery service required capabilities: " + err.Error())
	}

	newDSRequiredCapabilities, _, err := TOSession.GetDeliveryServiceRequiredCapabilities()
	if len(newDSRequiredCapabilities) > 0 {
		t.Errorf("get after delete returned capabilities")
	}
}
