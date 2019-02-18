package pluginsecuredbriak

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
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/apache/trafficcontrol/lib/go-log"
	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/config"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/riaksvc"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/securedb"

	"github.com/basho/riak-go-client"
)

const DeliveryServiceSSLKeysBucket = "ssl"
const DNSSECKeysBucket = "dnssec"
const DSSSLKeyVersionLatest = "latest"
const DefaultDSSSLKeyVersion = DSSSLKeyVersionLatest
const URLSigKeysBucket = "url_sig_keys"
const CDNURIKeysBucket = "cdn_uri_sig_keys"
const SSLKeysIndex = "sslkeys"
const CDNSSLKeysLimit = 1000 // TODO: emulates Perl; reevaluate?

type RiakConfig struct {
	// Port is the Riak API port; optional, the default will be used if it is missing.
	Port                  *uint  `json:"port"`
	User                  string `json:"user"`
	Pass                  string `json:"pass"`
	Insecure              bool   `json:"insecure"`
	HealthCheckIntervalMS uint64 `json:"health_check_interval_ms"`
}

// secureDB implements the securedb.SecureDB interface.
type secureDB struct {
	authOpts            *riak.AuthOptions
	riakPort            *uint
	healthCheckInterval time.Duration
}

func New(cfg RiakConfig, appCfg *config.Config) (securedb.SecureDB, error) {
	// TODO remove using old config in the next major version
	if cfg.Port == nil {
		cfg.Port = appCfg.RiakPort
	}
	if cfg.User == "" && appCfg.RiakAuthOptions != nil {
		cfg.User = appCfg.RiakAuthOptions.User
	}
	if cfg.Pass == "" && appCfg.RiakAuthOptions != nil {
		cfg.Pass = appCfg.RiakAuthOptions.Password
	}
	if cfg.Insecure == false && appCfg.RiakAuthOptions != nil && appCfg.RiakAuthOptions.TlsConfig != nil {
		cfg.Insecure = appCfg.RiakAuthOptions.TlsConfig.InsecureSkipVerify
	}
	if cfg.HealthCheckIntervalMS == 0 && riaksvc.HealthCheckInterval != 0 {
		cfg.HealthCheckIntervalMS = uint64(riaksvc.HealthCheckInterval / time.Millisecond)
	}

	db := &secureDB{
		authOpts: &riak.AuthOptions{
			User:     cfg.User,
			Password: cfg.Pass,
		},
		riakPort:            cfg.Port,
		healthCheckInterval: time.Duration(cfg.HealthCheckIntervalMS) * time.Millisecond}
	if cfg.Insecure {
		db.authOpts.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return db, nil
}

func (s *secureDB) Name() string { return "riak" }

func (s *secureDB) GetDeliveryServiceSSLKeysObjLatest(tx *sql.Tx, ds tc.DeliveryServiceName) (tc.DeliveryServiceSSLKeys, bool, error) {
	return s.GetDeliveryServiceSSLKeysObj(tx, ds, DSSSLKeyVersionLatest)
}

func (s *secureDB) GetDeliveryServiceSSLKeysObj(tx *sql.Tx, ds tc.DeliveryServiceName, version string) (tc.DeliveryServiceSSLKeys, bool, error) {
	key := tc.DeliveryServiceSSLKeys{}
	found := false
	err := WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		// get the deliveryservice ssl keys by xmlID and version
		ro, err := FetchObjectValues(makeDSSSLKeyKey(ds, version), DeliveryServiceSSLKeysBucket, cluster)
		if err != nil {
			return err
		}
		if len(ro) == 0 {
			return nil // not found
		}
		if err := json.Unmarshal(ro[0].Value, &key); err != nil {
			log.Errorf("failed at unmarshaling sslkey response: %s\n", err)
			return errors.New("unmarshalling Riak result: " + err.Error())
		}
		found = true
		return nil
	})
	if err != nil {
		return key, false, err
	}
	return key, found, nil
}

func (s *secureDB) PutDeliveryServiceSSLKeysObj(tx *sql.Tx, key tc.DeliveryServiceSSLKeys) error {
	keyJSON, err := json.Marshal(&key)
	if err != nil {
		return errors.New("marshalling key: " + err.Error())
	}
	err = WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		obj := &riak.Object{
			ContentType:     "application/json",
			Charset:         "utf-8",
			ContentEncoding: "utf-8",
			Key:             makeDSSSLKeyKey(tc.DeliveryServiceName(key.DeliveryService), key.Version.String()),
			Value:           []byte(keyJSON),
		}
		if err := SaveObject(obj, DeliveryServiceSSLKeysBucket, cluster); err != nil {
			return errors.New("saving Riak object: " + err.Error())
		}
		obj.Key = makeDSSSLKeyKey(tc.DeliveryServiceName(key.DeliveryService), DSSSLKeyVersionLatest)
		if err := SaveObject(obj, DeliveryServiceSSLKeysBucket, cluster); err != nil {
			return errors.New("saving Riak object: " + err.Error())
		}
		return nil
	})
	return err
}

func (s *secureDB) Ping(tx *sql.Tx) (tc.SDBPingResp, error) {
	servers, err := GetRiakServers(tx, s.riakPort)
	if err != nil {
		return tc.SDBPingResp{}, errors.New("getting riak servers: " + err.Error())
	}
	for _, server := range servers {
		cluster, err := GetRiakStorageCluster([]ServerAddr{server}, s.authOpts, s.healthCheckInterval)
		if err != nil {
			log.Errorf("RiakServersToCluster error for server %+v: %+v\n", server, err.Error())
			continue // try another server
		}
		if err = cluster.Start(); err != nil {
			log.Errorln("starting Riak cluster (for ping): " + err.Error())
			continue
		}
		if err := PingCluster(cluster); err != nil {
			if err := cluster.Stop(); err != nil {
				log.Errorln("stopping Riak cluster (after ping error): " + err.Error())
			}
			log.Errorf("Riak PingCluster error for server %+v: %+v\n", server, err.Error())
			continue
		}
		if err := cluster.Stop(); err != nil {
			log.Errorln("stopping Riak cluster (after ping success): " + err.Error())
		}
		return tc.SDBPingResp{Status: "OK", Server: server.FQDN + ":" + server.Port}, nil
	}
	return tc.SDBPingResp{}, errors.New("failed to ping any Riak server")
}

func (s *secureDB) GetDNSSECKeys(tx *sql.Tx, cdn tc.CDNName) (tc.DNSSECKeysSDB, bool, error) {
	key := tc.DNSSECKeysSDB{}
	found := false
	err := WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		ro, err := FetchObjectValues(string(cdn), DNSSECKeysBucket, cluster)
		if err != nil {
			return err
		}
		if len(ro) == 0 {
			return nil // not found
		}
		if err := json.Unmarshal(ro[0].Value, &key); err != nil {
			return errors.New("unmarshalling Riak dnssec response: " + err.Error())
		}
		found = true
		return nil
	})
	if err != nil {
		return key, false, err
	}
	return key, found, nil
}

func (s *secureDB) PutDNSSECKeys(tx *sql.Tx, keys tc.DNSSECKeysSDB, cdn tc.CDNName) error {
	keyJSON, err := json.Marshal(&keys)
	if err != nil {
		return errors.New("marshalling keys: " + err.Error())
	}

	err = WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		obj := &riak.Object{
			ContentType:     "application/json",
			Charset:         "utf-8",
			ContentEncoding: "utf-8",
			Key:             string(cdn),
			Value:           []byte(keyJSON),
		}
		if err = SaveObject(obj, DNSSECKeysBucket, cluster); err != nil {
			return errors.New("saving Riak object: " + err.Error())
		}
		return nil
	})
	return err
}

func (s *secureDB) GetBucketKey(tx *sql.Tx, bucket string, key string) ([]byte, bool, error) {
	val := []byte{}
	found := false
	err := WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		// get the deliveryservice ssl keys by xmlID and version
		ro, err := FetchObjectValues(key, bucket, cluster)
		if err != nil {
			return err
		}
		if len(ro) == 0 {
			return nil // not found
		}
		val = ro[0].Value
		found = true
		return nil
	})
	if err != nil {
		return val, false, err
	}
	return val, found, nil
}

func (s *secureDB) DeleteDSSSLKeys(tx *sql.Tx, ds tc.DeliveryServiceName, version string) error {
	err := WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		if err := DeleteObject(makeDSSSLKeyKey(ds, version), DeliveryServiceSSLKeysBucket, cluster); err != nil {
			return errors.New("deleting SSL keys: " + err.Error())
		}
		return nil
	})
	return err
}

func (s *secureDB) GetURLSigKeys(tx *sql.Tx, ds tc.DeliveryServiceName) (tc.URLSigKeys, bool, error) {
	val := tc.URLSigKeys{}
	found := false
	key := getURLSigConfigFileName(ds)
	err := WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		ro, err := FetchObjectValues(key, URLSigKeysBucket, cluster)
		if err != nil {
			return err
		}
		if len(ro) == 0 {
			return nil // not found
		}
		if err := json.Unmarshal(ro[0].Value, &val); err != nil {
			return errors.New("unmarshalling Riak response: " + err.Error())
		}
		found = true
		return nil
	})
	if err != nil {
		return val, false, err
	}
	return val, found, nil
}

func (s *secureDB) PutURLSigKeys(tx *sql.Tx, ds tc.DeliveryServiceName, keys tc.URLSigKeys) error {
	keyJSON, err := json.Marshal(&keys)
	if err != nil {
		return errors.New("marshalling keys: " + err.Error())
	}
	err = WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		obj := &riak.Object{
			ContentType:     "application/json",
			Charset:         "utf-8",
			ContentEncoding: "utf-8",
			Key:             getURLSigConfigFileName(ds),
			Value:           []byte(keyJSON),
		}
		if err = SaveObject(obj, URLSigKeysBucket, cluster); err != nil {
			return errors.New("saving Riak object: " + err.Error())
		}
		return nil
	})
	return err
}

func (s *secureDB) GetCDNSSLKeysObj(tx *sql.Tx, cdn tc.CDNName) ([]tc.CDNSSLKey, error) {
	keys := []tc.CDNSSLKey{}
	err := WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		// get the deliveryservice ssl keys by xmlID and version
		query := `cdn:` + string(cdn)
		filterQuery := `_yz_rk:*latest`
		searchDocs, err := Search(cluster, SSLKeysIndex, query, filterQuery, CDNSSLKeysLimit)
		if err != nil {
			return errors.New("riak search error: " + err.Error())
		}
		if len(searchDocs) == 0 {
			return nil // no error, and leave keys empty
		}
		keys = searchDocsToCDNSSLKeys(searchDocs)
		return nil
	})
	if err != nil {
		return nil, errors.New("with cluster error: " + err.Error())
	}
	return keys, nil
}

func (s *secureDB) GetURISigningKeys(tx *sql.Tx, ds tc.DeliveryServiceName) ([]byte, bool, error) {
	// TODO change to return actual object?
	key := ([]byte)(nil)
	found := false
	err := WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		ro, err := FetchObjectValues(string(ds), CDNURIKeysBucket, cluster)
		if err != nil {
			return err
		}

		if len(ro) > 0 {
			key = ro[0].Value
			found = true
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return key, found, nil
}

func makeDSSSLKeyKey(ds tc.DeliveryServiceName, version string) string {
	if version == "" {
		version = DefaultDSSSLKeyVersion
	}
	return string(ds) + "-" + version
}

// GetURLSigConfigFileName returns the filename of the Apache Traffic Server URLSig config file
// TODO move to ats config directory/file
func getURLSigConfigFileName(ds tc.DeliveryServiceName) string {
	return "url_sig_" + string(ds) + ".config"
}

// SearchDocsToCDNSSLKeys converts the SearchDoc array returned by Riak into a CDNSSLKey slice. If a SearchDoc doesn't contain expected fields, it creates the key with those fields defaulted to empty strings.
func searchDocsToCDNSSLKeys(docs []*riak.SearchDoc) []tc.CDNSSLKey {
	keys := []tc.CDNSSLKey{}
	for _, doc := range docs {
		key := tc.CDNSSLKey{}
		if dss := doc.Fields["deliveryservice"]; len(dss) > 0 {
			key.DeliveryService = dss[0]
		}
		if hosts := doc.Fields["hostname"]; len(hosts) > 0 {
			key.HostName = hosts[0]
		}
		if crts := doc.Fields["certificate.crt"]; len(crts) > 0 {
			key.Certificate.Crt = crts[0]
		}
		if keys := doc.Fields["certificate.key"]; len(keys) > 0 {
			key.Certificate.Key = keys[0]
		}
		keys = append(keys, key)
	}
	return keys
}

func (s *secureDB) DeleteCDNDNSSECKeys(tx *sql.Tx, cdn tc.CDNName) error {
	err := WithCluster(tx, s.authOpts, s.riakPort, s.healthCheckInterval, func(cluster StorageCluster) error {
		if err := DeleteObject(string(cdn), DNSSECKeysBucket, cluster); err != nil {
			return errors.New("deleting dnssec key '" + string(cdn) + "': " + err.Error())
		}
		return nil
	})
	return err
}
