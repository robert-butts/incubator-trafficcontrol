package plugin

// /*
//  * Licensed to the Apache Software Foundation (ASF) under one
//  * or more contributor license agreements.  See the NOTICE file
//  * distributed with this work for additional information
//  * regarding copyright ownership.  The ASF licenses this file
//  * to you under the Apache License, Version 2.0 (the
//  * "License"); you may not use this file except in compliance
//  * with the License.  You may obtain a copy of the License at
//  *
//  *   http://www.apache.org/licenses/LICENSE-2.0
//  *
//  * Unless required by applicable law or agreed to in writing,
//  * software distributed under the License is distributed on an
//  * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
//  * KIND, either express or implied.  See the License for the
//  * specific language governing permissions and limitations
//  * under the License.
//  */

// import (
// 	"database/sql"
// 	"strings"

// 	"github.com/apache/trafficcontrol/lib/go-log"
// 	"github.com/apache/trafficcontrol/lib/go-tc"
// 	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
// 	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
// 	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/securedb"
// )

// // NewSecureDBDemux takes multiple SecureDBs, and returns a single SecureDB, for which any function call will try all the input dbs, in order, and return the first success. When a SecureDB fails, an error is logged.
// // If len(dbs) == 0, nil is returned.
// func NewSecureDBDemux(dbs []securedb.SecureDB) securedb.SecureDB {
// 	if len(dbs) == 0 {
// 		return nil
// 	}
// 	return &secureDBDemux{dbs}
// }

// // secureDBDemux implements a securedb.SecureDB.
// // Note NewSecureDBDemux guarantees len(dbs) > 0
// type secureDBDemux struct {
// 	dbs []securedb.SecureDB
// }

// func (s *secureDBDemux) Name() string {
// 	names := []string{}
// 	for _, db := range s.dbs {
// 		names = append(names, db.Name())
// 	}
// 	return "demux[" + strings.Join(names, "--") + "]"
// }

// func (s *secureDBDemux) Ping(cfg *config.Config, pluginCfg map[string]interface{}, tx *sql.Tx) (resp tc.RiakPingResp, err error) {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		resp, err = db.Ping(cfg, pluginCfg, tx)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) GetDeliveryServiceSSLKeysObj(inf *api.APIInfo, ds tc.DeliveryServiceName, version string) (keys tc.DeliveryServiceSSLKeys, ok bool, err error) {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		keys, ok, err = db.GetDeliveryServiceSSLKeysObj(inf, ds, version)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) PutDeliveryServiceSSLKeysObj(inf *api.APIInfo, key tc.DeliveryServiceSSLKeys) (err error) {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		err = db.PutDeliveryServiceSSLKeysObj(inf, key)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) GetDNSSECKeys(inf *api.APIInfo, cdn tc.CDNName) (resp tc.DNSSECKeysRiak, ok bool, err error) {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		resp, ok, err = db.Ping(cfg, pluginCfg, tx)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) PutDNSSECKeys(inf *api.APIInfo, keys tc.DNSSECKeysRiak, cdn tc.CDNName) error {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		err = db.PutDNSSECKeys(cfg, pluginCfg, tx)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) GetBucketKey(inf *api.APIInfo, bucket string, key string) (bts []byte, ok bool, err error) {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		bts, ok, err = db.GetBucketKey(inf, bucket, key)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) DeleteDSSSLKeys(inf *api.APIInfo, ds tc.DeliveryServiceName, version string) error {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		err = db.DeleteDSSSLKeys(inf, ds, version)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) GetURLSigKeys(inf *api.APIInfo, ds tc.DeliveryServiceName) (resp tc.URLSigKeys, ok bool, err error) {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		resp, ok, err = db.GetURLSigKeys(inf, ds)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) PutURLSigKeys(inf *api.APIInfo, ds tc.DeliveryServiceName, keys tc.URLSigKeys) error {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		err = db.PutURLSigKeys(inf, ds, keys)
// 		return err
// 	})
// 	return
// }

// func (s *secureDBDemux) GetCDNSSLKeysObj(inf *api.APIInfo, cdn tc.CDNName) (keys []tc.CDNSSLKey, err error) {
// 	firstSuccessOf(s.dbs, func(db securedb.SecureDB) error {
// 		keys, err = db.GetCDNSSLKeysObj(inf, cdn)
// 		return err
// 	})
// }

// // firstSuccessOf calls each f with each members of dbs, and stops after the first nil error.
// func firstSuccessOf(dbs []securedb.SecureDB, f func(db securedb.SecureDB) error) {
// 	return firstSuccess(errFuncs(db, f))
// }

// // dbsToErrFuncs takes an array of dbs, and a function to operate on the db and return an error, and returns an array of firstSuccess funcs.
// // This is designed for f to have side effects, such that firstSuccess will call f and stop after the first success.
// func errFuncs(dbs []securedb.SecureDB, f func(db securedb.SecureDB) error) []func() error {
// 	fs := []func() error{}
// 	for _, db := range dbs {
// 		fs = append(fs, func() error {
// 			return f(db)
// 		})
// 	}
// 	return fs
// }

// // firstSuccess executes the given functions, stopping on the first func to return a nil error. If all funcs return non-nil errors, the first function is called again.
// // The idea is, if every func fails, you probably want to return the initial error.
// // This function exists for demultiplexing a series of objects. This is the best I could come up with, with Go's weak type system and lack of metaprogramming.
// func firstSuccess(fs []func() error) {
// 	if len(fs) == 0 {
// 		return
// 	}
// 	for _, f := range fs {
// 		if err := f(); err == nil {
// 			return
// 		}
// 	}
// 	fs[0]()
// }
