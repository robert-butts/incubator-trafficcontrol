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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/iplugin"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/securedb"
)

// List returns the list of plugins compiled into the calling executable.
func List() []string {
	l := []string{}
	for _, p := range initPlugins {
		l = append(l, p.name)
	}
	return l
}

func Get(appCfg *config.Config, db *sql.DB) (iplugin.Plugins, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, errors.New("starting transaction: " + err.Error())
	}
	defer tx.Commit()

	log.Infof("plugin.Get given: %+v\n", appCfg.Plugins)
	pluginSlice := getEnabled(appCfg.Plugins)
	pluginCfg := loadConfig(pluginSlice, appCfg.PluginConfig)
	secureDB := loadSecureDB(pluginSlice, pluginCfg, appCfg, tx)
	ctx := map[string]*interface{}{}
	return plugins{slice: pluginSlice, cfg: pluginCfg, ctx: ctx, sDB: secureDB}, nil
}

func getEnabled(enabled []string) pluginsSlice {
	enabledM := map[string]struct{}{}
	for _, name := range enabled {
		enabledM[name] = struct{}{}
	}
	enabledPlugins := pluginsSlice{}
	for _, plugin := range initPlugins {
		if _, ok := enabledM[plugin.name]; !ok {
			log.Infoln("getEnabled skipping: '" + plugin.name + "'")
			continue
		}
		log.Infoln("plugin enabling: '" + plugin.name + "'")
		enabledPlugins = append(enabledPlugins, plugin)
	}
	sort.Sort(enabledPlugins)
	return enabledPlugins
}

func loadConfig(ps pluginsSlice, configJSON map[string]json.RawMessage) map[string]interface{} {
	pluginConfigLoaders := loadFuncs(ps)
	cfg := make(map[string]interface{}, len(configJSON))
	for name, b := range configJSON {
		if loadF := pluginConfigLoaders[name]; loadF != nil {
			cfg[name] = loadF(b)
		}
	}
	return cfg
}

func loadFuncs(ps pluginsSlice) map[string]LoadFunc {
	lf := map[string]LoadFunc{}
	for _, plugin := range ps {
		if plugin.funcs.load == nil {
			continue
		}
		lf[plugin.name] = LoadFunc(plugin.funcs.load)
	}
	return lf
}

func loadSecureDB(ps pluginsSlice, cfg map[string]interface{}, appCfg *config.Config, tx *sql.Tx) securedb.SecureDB {
	for _, plugin := range ps {
		if plugin.funcs.loadSecureDB == nil {
			continue
		}

		dat := iplugin.LoadSecureDBData{Cfg: cfg[plugin.name], AppCfg: appCfg, Tx: tx}
		db, err := plugin.funcs.loadSecureDB(dat)
		if err != nil {
			log.Errorln("failed to load secure db plugin '" + plugin.name + "': " + err.Error())
			continue
		}
		if _, err := db.Ping(tx); err != nil {
			log.Errorln("failed to load secure db plugin '" + plugin.name + "': ping: " + err.Error())
			continue
		}
		return db
	}
	return nil
}

func AddPlugin(priority uint64, funcs Funcs) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		fmt.Println(time.Now().Format(time.RFC3339Nano) + " Error plugin.AddPlugin: runtime.Caller failed, can't get plugin names") // print, because this is called in init, loggers don't exist yet
		os.Exit(1)
	}

	pluginName := strings.TrimSuffix(path.Base(filename), ".go")
	log.Debugln("AddPlugin adding " + pluginName)
	initPlugins = append(initPlugins, pluginObj{funcs: funcs, priority: priority, name: pluginName})
}

type Funcs struct {
	load LoadFunc
	// loadSecureDB is called once on startup, like load. If multiple plugins have loadSecureDB hooks, they are called in priority order, and the first with a successful Ping is used. If all fail, the first priority is used, and an error is logged.
	loadSecureDB LoadSecureDBFunc
	onStartup    StartupFunc
	onRequest    OnRequestFunc
}

type IsRequestHandled bool

const (
	RequestHandled   = IsRequestHandled(true)
	RequestUnhandled = IsRequestHandled(false)
)

type LoadFunc func(json.RawMessage) interface{}
type StartupFunc func(iplugin.StartupData)
type OnRequestFunc func(iplugin.OnRequestData) IsRequestHandled
type LoadSecureDBFunc func(iplugin.LoadSecureDBData) (securedb.SecureDB, error)

type pluginObj struct {
	funcs    Funcs
	priority uint64
	name     string
}

type plugins struct {
	slice pluginsSlice
	cfg   map[string]interface{}
	ctx   map[string]*interface{}
	sDB   securedb.SecureDB
}

type pluginsSlice []pluginObj

func (p pluginsSlice) Len() int           { return len(p) }
func (p pluginsSlice) Less(i, j int) bool { return p[i].priority < p[j].priority }
func (p pluginsSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// initPlugins is where plugins are registered via their init functions.
var initPlugins = pluginsSlice{}

func (ps plugins) OnStartup(d iplugin.StartupData) {
	for _, p := range ps.slice {
		ictx := interface{}(nil)
		ps.ctx[p.name] = &ictx
		if p.funcs.onStartup == nil {
			continue
		}
		d.Ctx = ps.ctx[p.name]
		d.Cfg = ps.cfg[p.name]
		p.funcs.onStartup(d)
	}
}

// OnRequest returns a boolean whether to immediately stop processing the request. If a plugin returns true, this is immediately returned with no further plugins processed.
func (ps plugins) OnRequest(d iplugin.OnRequestData) bool {
	log.Debugf("DEBUG plugins.OnRequest calling %+v plugins\n", len(ps.slice))
	for _, p := range ps.slice {
		if p.funcs.onRequest == nil {
			log.Debugln("plugins.OnRequest plugging " + p.name + " - no onRequest func")
			continue
		}
		d.Ctx = ps.ctx[p.name]
		d.Cfg = ps.cfg[p.name]
		log.Debugln("plugins.OnRequest plugging " + p.name)
		if stop := p.funcs.onRequest(d); stop {
			return true
		}
	}
	return false
}

// SecureDB returns a securedb.SecureDB which is a demultiplexed version of all securedb plugins. If no plugins were loaded successfully, this returns nil.
func (ps plugins) SecureDB() securedb.SecureDB { return ps.sDB }
