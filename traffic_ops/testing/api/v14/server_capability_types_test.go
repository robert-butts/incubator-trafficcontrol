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

func TestServerCapabilityTypes(t *testing.T) {
	WithObjs(t, []TCObj{ServerCapabilityTypes}, func() {
		// test create, delete, which also tests get; capability types have no put/update
	})
}

func CreateTestServerCapabilityTypes(t *testing.T) {
	types := []string{"CACHE_MEMORY", "CACHE_DISK"}
	for _, tp := range types {
		_, _, err := TOSession.CreateServerCapabilityType(tp)
		if err != nil {
			t.Errorf("could not CREATE server capability type: %v\n", err)
		}
	}

	serverCapabilityTypes, _, err := TOSession.GetServerCapabilityTypes()
	if err != nil {
		t.Fatalf("getting server capability types: " + err.Error())
	}
	if len(serverCapabilityTypes) != len(types) {
		t.Errorf("getting server capability types: expected %v, actual: 0", len(types))
	}
}

func DeleteTestServerCapabilityTypes(t *testing.T) {
	serverCapabilityTypes, _, err := TOSession.GetServerCapabilityTypes()
	if err != nil {
		t.Fatalf("getting server capability types: " + err.Error())
	}
	if len(serverCapabilityTypes) < 1 {
		t.Errorf("getting server capability types: expected > 1, actual: 0")
	}

	for _, tp := range serverCapabilityTypes {
		_, _, err = TOSession.DeleteServerCapabilityType(*tp.Name)
		if err != nil {
			t.Errorf("deleting server capability type: " + err.Error())
		}
	}

	serverCapabilityTypes, _, err = TOSession.GetServerCapabilityTypes()
	if err != nil {
		t.Fatalf("getting server capability types: " + err.Error())
	}
	if len(serverCapabilityTypes) != 0 {
		t.Errorf("getting server capability types after deletion: expected 0, actual: %v", len(serverCapabilityTypes))
	}
}
