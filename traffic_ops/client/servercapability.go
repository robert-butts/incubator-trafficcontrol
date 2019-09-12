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

package client

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

const ServerCapabilityTypesPath = apiBase + "/server_capability_types"

func (to *Session) CreateServerCapabilityType(serverCapabilityType string) (tc.Alerts, ReqInf, error) {
	remoteAddr := net.Addr(nil)
	reqBody, err := json.Marshal(tc.ServerCapabilityType{Name: &serverCapabilityType})
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return tc.Alerts{}, reqInf, err
	}
	resp, remoteAddr, err := to.request(http.MethodPost, ServerCapabilityTypesPath, reqBody)
	if err != nil {
		return tc.Alerts{}, reqInf, err
	}
	defer resp.Body.Close()
	alerts := tc.Alerts{}
	err = json.NewDecoder(resp.Body).Decode(&alerts)
	return alerts, reqInf, nil
}

func (to *Session) GetServerCapabilityTypes() ([]tc.ServerCapabilityType, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, ServerCapabilityTypesPath, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return nil, reqInf, err
	}
	defer resp.Body.Close()

	data := struct {
		Response []tc.ServerCapabilityType `json:"response"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	return data.Response, reqInf, nil
}

func (to *Session) DeleteServerCapabilityType(serverCapabilityType string) (tc.Alerts, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodDelete, ServerCapabilityTypesPath+`/`+serverCapabilityType, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return tc.Alerts{}, reqInf, err
	}
	defer resp.Body.Close()
	alerts := tc.Alerts{}
	if err = json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return tc.Alerts{}, reqInf, errors.New("decoding response: " + err.Error())
	}
	return alerts, reqInf, nil
}

func (to *Session) GetServerCapabilities() ([]tc.ServerCapability, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, apiBase+`/servers_capabilities`, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return nil, reqInf, err
	}
	defer resp.Body.Close()
	data := struct {
		Response []tc.ServerCapability `json:"response"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, reqInf, errors.New("decoding response: " + err.Error())
	}
	return data.Response, reqInf, nil
}

func (to *Session) GetServerCapabilitiesByServerID(serverID int) (tc.ServerCapability, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, apiBase+`/servers/`+strconv.Itoa(serverID)+`/capabilities`, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return tc.ServerCapability{}, reqInf, err
	}
	defer resp.Body.Close()
	data := struct {
		Response []tc.ServerCapability `json:"response"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return tc.ServerCapability{}, reqInf, errors.New("decoding response: " + err.Error())
	}
	if len(data.Response) == 0 {
		return tc.ServerCapability{}, reqInf, errors.New("server returned no error, but no response")
	}
	return data.Response[0], reqInf, nil
}

func (to *Session) UpdateServerCapabilities(serverID int, capabilities []string) (tc.Alerts, ReqInf, error) {
	remoteAddr := net.Addr(nil)
	reqBody, err := json.Marshal(tc.ServerCapability{ID: &serverID, Capabilities: capabilities})
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return tc.Alerts{}, reqInf, err
	}
	resp, remoteAddr, err := to.request(http.MethodPut, apiBase+`/servers/`+strconv.Itoa(serverID)+`/capabilities`, reqBody)
	if err != nil {
		return tc.Alerts{}, reqInf, err
	}
	defer resp.Body.Close()
	alerts := tc.Alerts{}
	if err = json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return tc.Alerts{}, reqInf, errors.New("decoding response: " + err.Error())
	}
	return alerts, reqInf, nil
}

func (to *Session) GetDeliveryServiceRequiredCapabilities() ([]tc.DeliveryServiceRequiredCapability, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, apiBase+`/deliveryservices_required_capabilities`, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return nil, reqInf, err
	}
	defer resp.Body.Close()
	data := struct {
		Response []tc.DeliveryServiceRequiredCapability `json:"response"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, reqInf, errors.New("decoding response: " + err.Error())
	}
	return data.Response, reqInf, nil
}

func (to *Session) GetDeliveryServiceRequiredCapabilitiesByDeliveryServiceID(deliveryServiceID int) (tc.DeliveryServiceRequiredCapability, ReqInf, error) {
	resp, remoteAddr, err := to.request(http.MethodGet, apiBase+`/deliveryservices/`+strconv.Itoa(deliveryServiceID)+`/required_capabilities`, nil)
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return tc.DeliveryServiceRequiredCapability{}, reqInf, err
	}
	defer resp.Body.Close()
	data := struct {
		Response []tc.DeliveryServiceRequiredCapability `json:"response"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return tc.DeliveryServiceRequiredCapability{}, reqInf, errors.New("decoding response: " + err.Error())
	}
	if len(data.Response) == 0 {
		return tc.DeliveryServiceRequiredCapability{}, reqInf, errors.New("server returned no error, but no response")
	}
	return data.Response[0], reqInf, nil
}

func (to *Session) UpdateDeliveryServiceRequiredCapabilities(deliveryServiceID int, capabilities []string) (tc.Alerts, ReqInf, error) {
	remoteAddr := (net.Addr)(nil)
	reqBody, err := json.Marshal(tc.DeliveryServiceRequiredCapability{ID: &deliveryServiceID, Capabilities: capabilities})
	reqInf := ReqInf{CacheHitStatus: CacheHitStatusMiss, RemoteAddr: remoteAddr}
	if err != nil {
		return tc.Alerts{}, reqInf, err
	}
	resp, remoteAddr, err := to.request(http.MethodPut, apiBase+`/deliveryservices/`+strconv.Itoa(deliveryServiceID)+`/required_capabilities`, reqBody)
	if err != nil {
		return tc.Alerts{}, reqInf, err
	}
	defer resp.Body.Close()
	alerts := tc.Alerts{}
	if err = json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return tc.Alerts{}, reqInf, errors.New("decoding response: " + err.Error())
	}
	return alerts, reqInf, nil
}
