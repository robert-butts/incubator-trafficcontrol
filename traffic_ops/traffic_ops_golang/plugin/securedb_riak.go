package plugin

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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/iplugin"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/plugin/securedb_riak"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/securedb"
)

func init() {
	AddPlugin(10000, Funcs{load: riakLoad, loadSecureDB: riakLoadSecureDB})
}

func riakLoad(b json.RawMessage) interface{} {
	cfg := pluginsecuredbriak.RiakConfig{}
	err := json.Unmarshal(b, &cfg)
	if err != nil {
		log.Errorln(`Failed to unmarshal securedb_riak plugin config. Config should look like: {"plugin_config": {"securedb_riak_config":{"port": 8087, "user":"myUser", "pass": "myPass", "health_check_interval_ms": 5000, "insecure": false}}}`)
		return nil
	}
	log.Infoln("Plugin securedb_riak successfully loaded config!")
	return &cfg
}

func riakLoadSecureDB(d iplugin.LoadSecureDBData) (securedb.SecureDB, error) {
	if d.Cfg == nil {
		return nil, errors.New("no config")
	}
	cfg, ok := d.Cfg.(*pluginsecuredbriak.RiakConfig)
	if !ok {
		// should never happen
		return nil, fmt.Errorf("config '%v' type '%T' expected *RiakConfig\n", d.Cfg, d.Cfg)
	}
	if cfg == nil {
		return nil, errors.New("nil config")
	}
	return pluginsecuredbriak.New(*cfg, d.AppCfg)
}
