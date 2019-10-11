package main

import (
	"encoding/json"

	"github.com/apache/trafficcontrol/lib/go-tc"
)

func FakeCRConfig() *tc.CRConfig {
	jstr := `
{
  "config": {
    "api.cache-control.max-age": "10",
    "api.port": "3333",
    "certificates.polling.interval": "300000",
    "consistent.dns.routing": "true",
    "coveragezone.polling.interval": "3600000",
    "coveragezone.polling.url": "http://ipcdn-tools-03.cdnlab.comcast.net/ipcdn/CZF/current/comcast_ipcdn_czf-current.json",
    "dnssec.dynamic.response.expiration": "300s",
    "domain_name": "top.comcast.net",
    "edge.dns.limit": "6",
    "edge.dns.routing": "true",
    "edge.http.limit": "6",
    "edge.http.routing": "true",
    "federationmapping.polling.interval": "60000",
    "federationmapping.polling.url": "https://${toHostname}/internal/api/1.2/federations.json",
    "geolocation.polling.interval": "86400000",
    "geolocation.polling.url": "http://ipcdn-tools-03.cdnlab.comcast.net/MaxMind/auto/GeoIP2-City.mmdb.gz",
    "keystore.maintenance.interval": "300",
    "location": "/opt/trafficserver/etc/trafficserver",
    "monitor:///opt/tomcat/logs/access.log": "index=index_odol_test;sourcetype=access_ccr",
    "neustar.polling.interval": "86400000",
    "neustar.polling.url": "http://ipcdn-tools-03.cdnlab.comcast.net/neustar/latest.tar.gz",
    "over-the-top.dnssec.inception": "1450219172",
    "soa": {
      "admin": "twelve_monkeys",
      "expire": "604800",
      "minimum": "30",
      "refresh": "28800",
      "retry": "7200"
    },
    "steeringmapping.polling.interval": "60000",
    "ttls": {
      "A": "3600",
      "AAAA": "3600",
      "DNSKEY": "30",
      "DS": "30",
      "NS": "3600",
      "SOA": "86400"
    },
    "upgrade_http_routing_name": "ccr",
    "zonemanager.cache.maintenance.interval": "300",
    "zonemanager.threadpool.scale": "0.50"
  },
  "contentServers": {
    "ec0": {
      "cacheGroup": "cg-ec0",
      "fqdn": "ec0.test",
      "hashCount": 1000,
      "hashId": "ec0",
      "httpsPort": 30001,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "ip6": "::1/64",
      "locationId": "30001",
      "port": 20001,
      "profile": "pr-ec0",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec1": {
      "cacheGroup": "cg-ec1",
      "fqdn": "ec1.test",
      "hashCount": 1000,
      "hashId": "ec1",
      "httpsPort": 30002,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30002",
      "port": 20002,
      "profile": "pr-ec1",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec2": {
      "cacheGroup": "cg-ec2",
      "fqdn": "ec2.test",
      "hashCount": 1000,
      "hashId": "ec2",
      "httpsPort": 30003,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30003",
      "port": 20003,
      "profile": "pr-ec2",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec3": {
      "cacheGroup": "cg-ec3",
      "fqdn": "ec3.test",
      "hashCount": 1000,
      "hashId": "ec3",
      "httpsPort": 30004,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30004",
      "port": 20004,
      "profile": "pr-ec3",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec4": {
      "cacheGroup": "cg-ec4",
      "fqdn": "ec4.test",
      "hashCount": 1000,
      "hashId": "ec4",
      "httpsPort": 30005,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30005",
      "port": 20005,
      "profile": "pr-ec4",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec5": {
      "cacheGroup": "cg-ec5",
      "fqdn": "ec5.test",
      "hashCount": 1000,
      "hashId": "ec5",
      "httpsPort": 30006,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30006",
      "port": 20006,
      "profile": "pr-ec5",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec6": {
      "cacheGroup": "cg-ec6",
      "fqdn": "ec6.test",
      "hashCount": 1000,
      "hashId": "ec6",
      "httpsPort": 30007,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30007",
      "port": 20007,
      "profile": "pr-ec6",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec7": {
      "cacheGroup": "cg-ec7",
      "fqdn": "ec7.test",
      "hashCount": 1000,
      "hashId": "ec7",
      "httpsPort": 30008,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30008",
      "port": 20008,
      "profile": "pr-ec7",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec8": {
      "cacheGroup": "cg-ec8",
      "fqdn": "ec8.test",
      "hashCount": 1000,
      "hashId": "ec8",
      "httpsPort": 30009,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30009",
      "port": 20009,
      "profile": "pr-ec8",
      "status": "REPORTED",
      "type": "EDGE"
    },
    "ec9": {
      "cacheGroup": "cg-ec9",
      "fqdn": "ec9.test",
      "hashCount": 1000,
      "hashId": "ec9",
      "httpsPort": 30010,
      "interfaceName": "bond0",
      "ip": "127.0.0.1",
      "locationId": "30010",
      "port": 20010,
      "profile": "pr-ec9",
      "status": "REPORTED",
      "type": "EDGE"
    }
  },
  "contentRouters": {
    "tr0": {
      "apiPort": "3333",
      "fqdn": "tr0.test",
      "ip": "127.0.0.1",
      "ip6": "::1",
      "location": "cg-tr-0",
      "port": 40001,
      "profile": "pr-tr-0",
      "status": "ONLINE",
      "hashCount": 999
    },
    "tr1": {
      "apiPort": "3333",
      "fqdn": "tr1.test",
      "ip": "127.0.0.1",
      "ip6": "::1",
      "location": "cg-tr-1",
      "port": 40001,
      "profile": "pr-tr-1",
      "status": "ONLINE",
      "hashCount": 999
    }
  },
  "deliveryServices": {
    "ds0": {
      "coverageZoneOnly": "false",
      "dispersion": {
        "limit": 1,
        "shuffled": "true"
      },
      "domains": [
        "ds0.cdn0.test"
      ],
      "geoLocationProvider": "maxmindGeolocationService",
      "matchSets": [
        {
          "protocol": "HTTP",
          "matchList": [
            {
              "regex": ".*\\.ds0\\..*",
              "match-type": "HOST"
            }
          ]
        }
      ],
      "missLocation": {
        "latitude": 41.0,
        "longitude": -90.0
      },
      "protocol": {
        "acceptHttp": "true",
        "acceptHttps": "false",
        "redirectOnHttps": "false"
      },
      "regionalGeoBlocking": "false",
      "soa": {
        "admin": "traffic_ops",
        "expire": "604800",
        "minimum": "30",
        "refresh": "28800",
        "retry": "7200"
      },
      "sslEnabled": "false",
      "ttl": 30,
      "ttls": {
        "A": "30",
        "AAAA": "30",
        "NS": "3600",
        "SOA": "86400"
      },
      "maxDnsIpsForLocation": 3,
      "ip6RoutingEnabled": "true",
      "routingName": "edge"
    },
    "ds1": {
      "coverageZoneOnly": "false",
      "dispersion": {
        "limit": 1,
        "shuffled": "true"
      },
      "domains": [
        "ds1.cdn0.test"
      ],
      "geoLocationProvider": "maxmindGeolocationService",
      "matchSets": [
        {
          "protocol": "DNS",
          "matchList": [
            {
              "regex": ".*\\.ds1\\..*",
              "match-type": "HOST"
            }
          ]
        }
      ],
      "missLocation": {
        "latitude": 41.0,
        "longitude": -89.0
      },
      "protocol": {
        "acceptHttp": "true",
        "acceptHttps": "false",
        "redirectOnHttps": "false"
      },
      "regionalGeoBlocking": "false",
      "soa": {
        "admin": "traffic_ops",
        "expire": "604800",
        "minimum": "30",
        "refresh": "28800",
        "retry": "7200"
      },
      "sslEnabled": "false",
      "ttl": 30,
      "ttls": {
        "A": "30",
        "AAAA": "30",
        "NS": "3600",
        "SOA": "86400"
      },
      "maxDnsIpsForLocation": 3,
      "ip6RoutingEnabled": "true",
      "routingName": "edge",
      "bypassDestination": {
        "DNS": {
          "ttl": 30
        }
      }
    }
  },
  "edgeLocations": {
    "cg-ec0": {
      "latitude": 47.871234,
      "longitude": -124.123456
    },
    "cg-ec1": {
      "latitude": 38.123456,
      "longitude": -73.123456
    },
    "cg-ec2": {
      "latitude": 40.123456,
      "longitude": -73.123456
    },
    "cg-ec3": {
      "latitude": 38.123456,
      "longitude": -74.123456
    },
    "cg-ec4": {
      "latitude": 39.123456,
      "longitude": -73.123456
    },
    "cg-ec5": {
      "latitude": 39.123456,
      "longitude": -74.123456
    },
    "cg-ec6": {
      "latitude": 37.123456,
      "longitude": -72.123456
    },
    "cg-ec7": {
      "latitude": 38.123456,
      "longitude": -70.123456
    },
    "cg-ec8": {
      "latitude": 39.123456,
      "longitude": -74.123456
    },
    "cg-ec9": {
      "latitude": 39.123456,
      "longitude": -75.123456
    }
  },
  "trafficRouterLocations": {
    "cg-tr-0": {
      "latitude": 39.123456,
      "longitude": -78.123456
    },
    "cg-tr-1": {
      "latitude": 37.123456,
      "longitude": -121.123456
    }
  },
  "monitors": {
    "tm0": {
      "fqdn": "tm0.test",
      "ip": "127.0.0.1",
      "location": "cg-tm0",
      "port": 20001,
      "profile": "pr-tm-0",
      "status": "ONLINE"
    },
    "tm1": {
      "fqdn": "tm1.test",
      "ip": "127.0.0.1",
      "location": "cg-tm1",
      "port": 20002,
      "profile": "pr-tm-1",
      "status": "ONLINE"
    },
    "tm2": {
      "fqdn": "tm2.test",
      "ip": "127.0.0.1",
      "location": "cg-tm2",
      "port": 20002,
      "profile": "pr-tm2",
      "status": "ONLINE"
    },
    "tm3": {
      "fqdn": "tm3.test",
      "ip": "127.0.0.1",
      "location": "cg-tm3",
      "port": 20002,
      "profile": "pr-tm3",
      "status": "ONLINE"
    },
    "tm4": {
      "fqdn": "tm1.test",
      "ip": "127.0.0.1",
      "location": "cg-tm4",
      "port": 20002,
      "profile": "pr-tm4",
      "status": "ONLINE"
    },
    "tm5": {
      "fqdn": "tm5.test",
      "ip": "127.0.0.1",
      "location": "cg-tm5",
      "port": 20002,
      "profile": "pr-tm5",
      "status": "ONLINE"
    },
    "tm6": {
      "fqdn": "tm6.test",
      "ip": "127.0.0.1",
      "location": "cg-tm6",
      "port": 20002,
      "profile": "pr-tm6",
      "status": "ONLINE"
    },
    "tm7": {
      "fqdn": "tm7.test",
      "ip": "127.0.0.1",
      "location": "cg-tm7",
      "port": 20002,
      "profile": "pr-tm7",
      "status": "ONLINE"
    },
    "tm8": {
      "fqdn": "tm8.test",
      "ip": "127.0.0.1",
      "location": "cg-tm8",
      "port": 20002,
      "profile": "pr-tm8",
      "status": "ONLINE"
    },
    "tm9": {
      "fqdn": "tm9.test",
      "ip": "127.0.0.1",
      "location": "cg-tm9",
      "port": 20002,
      "profile": "pr-tm9",
      "status": "ONLINE"
    }
  },
  "stats": {
    "CDN_name": "cdn0",
    "date": 1521570101,
    "tm_host": "cdn0.test",
    "tm_path": "/generate/crconfig",
    "tm_user": "me",
    "tm_version": "0.1.golang"
  }
}
`
	crc := tc.CRConfig{}
	if err := json.Unmarshal([]byte(jstr), &crc); err != nil {
		panic(err)
	}
	return &crc
}

func FakeMonitoring() *TrafficMonitorConfig2 {
	jstr := `
{
    "trafficServers": [
      {
        "profile": "pr-ec0",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20001,
        "cacheGroup": "cg-ec0",
        "hostname": "ec0",
        "fqdn": "ec0.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec0"
      },
      {
        "profile": "pr-ec1",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20002,
        "cacheGroup": "cg-ec1",
        "hostname": "ec1",
        "fqdn": "ec1.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec1"
      },
      {
        "profile": "pr-ec2",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20003,
        "cacheGroup": "cg-ec2",
        "hostname": "ec2",
        "fqdn": "ec2.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec2"
      },
      {
        "profile": "pr-ec3",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20004,
        "cacheGroup": "cg-ec3",
        "hostname": "ec3",
        "fqdn": "ec3.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec3"
      },
      {
        "profile": "pr-ec4",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20005,
        "cacheGroup": "cg-ec4",
        "hostname": "ec4",
        "fqdn": "ec4.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec4"
      },
      {
        "profile": "pr-ec5",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20006,
        "cacheGroup": "cg-ec5",
        "hostname": "ec5",
        "fqdn": "ec5.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec5"
      },
      {
        "profile": "pr-ec6",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20007,
        "cacheGroup": "cg-ec6",
        "hostname": "ec6",
        "fqdn": "ec6.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec6"
      },
      {
        "profile": "pr-ec7",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20008,
        "cacheGroup": "cg-ec7",
        "hostname": "ec7",
        "fqdn": "ec7.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec7"
      },
      {
        "profile": "pr-ec8",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20009,
        "cacheGroup": "cg-ec8",
        "hostname": "ec8",
        "fqdn": "ec8.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec8"
      },
      {
        "profile": "pr-ec9",
        "status": "REPORTED",
        "ip": "127.0.0.1",
        "ip6": "::1/64",
        "port": 20010,
        "cacheGroup": "cg-ec9",
        "hostname": "ec9",
        "fqdn": "ec9.test",
        "interfacename": "bond0",
        "type": "EDGE",
        "hashid": "ec9"
      }
    ],
    "trafficMonitors": [
      {
        "profile": "pr-tm0",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20001,
        "cachegroup": "cg-tm0",
        "hostname": "tm0",
        "fqdn": "tm0.test"
      },
      {
        "profile": "pr-tm1",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20002,
        "cachegroup": "cg-tm1",
        "hostname": "tm1",
        "fqdn": "tm1.test"
      },
      {
        "profile": "pr-tm2",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20003,
        "cachegroup": "cg-tm2",
        "hostname": "tm2",
        "fqdn": "tm2.test"
      },
      {
        "profile": "pr-tm3",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20004,
        "cachegroup": "cg-tm3",
        "hostname": "tm3",
        "fqdn": "tm3.test"
      },
      {
        "profile": "pr-tm4",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20005,
        "cachegroup": "cg-tm4",
        "hostname": "tm4",
        "fqdn": "tm4.test"
      },
      {
        "profile": "pr-tm5",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20006,
        "cachegroup": "cg-tm5",
        "hostname": "tm5",
        "fqdn": "tm5.test"
      },
      {
        "profile": "pr-tm6",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20007,
        "cachegroup": "cg-tm6",
        "hostname": "tm6",
        "fqdn": "tm6.test"
      },
      {
        "profile": "pr-tm7",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20008,
        "cachegroup": "cg-tm7",
        "hostname": "tm7",
        "fqdn": "tm7.test"
      },
      {
        "profile": "pr-tm8",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20009,
        "cachegroup": "cg-tm8",
        "hostname": "tm8",
        "fqdn": "tm8.test"
      },
      {
        "profile": "pr-tm9",
        "status": "ONLINE",
        "ip": "127.0.0.1",
        "ip6": "",
        "port": 20010,
        "cachegroup": "cg-tm9",
        "hostname": "tm9",
        "fqdn": "tm9.test"
      }
    ],
    "cacheGroups": [
      {
        "name": "cg-ec0",
        "coordinates": {
          "latitude": 37.123456,
          "longitude": -121.123456
        }
      },
      {
        "name": "cg-ec1",
        "coordinates": {
          "latitude": 38.123456,
          "longitude": -121.123456
        }
      },
      {
        "name": "cg-ec2",
        "coordinates": {
          "latitude": 39.123456,
          "longitude": -122.123456
        }
      },
      {
        "name": "cg-ec3",
        "coordinates": {
          "latitude": 38.123456,
          "longitude": -122.123456
        }
      },
      {
        "name": "cg-ec4",
        "coordinates": {
          "latitude": 36.123456,
          "longitude": -121.123456
        }
      },
      {
        "name": "cg-ec5",
        "coordinates": {
          "latitude": 37.123456,
          "longitude": -120.123456
        }
      },
      {
        "name": "cg-ec6",
        "coordinates": {
          "latitude": 36.123456,
          "longitude": -120.123456
        }
      },
      {
        "name": "cg-ec7",
        "coordinates": {
          "latitude": 37.123456,
          "longitude": -123.123456
        }
      },
      {
        "name": "cg-ec8",
        "coordinates": {
          "latitude": 36.123456,
          "longitude": -120.123456
        }
      },
      {
        "name": "cg-ec9",
        "coordinates": {
          "latitude": 38.123456,
          "longitude": -123.123456
        }
      },
      {
        "name": "cg-tr0",
        "coordinates": {
          "latitude": 37.123456,
          "longitude": -121.123456
        }
      },
      {
        "name": "cg-tr1",
        "coordinates": {
          "latitude": 42.123456,
          "longitude": -71.123456
        }
      },
      {
        "name": "cg-tm0",
        "coordinates": {
          "latitude": 37.123456,
          "longitude": -121.123456
        }
      },
      {
        "name": "cg-tm1",
        "coordinates": {
          "latitude": 36.123456,
          "longitude": -121.123456
        }
      },
      {
        "name": "cg-tm2",
        "coordinates": {
          "latitude": 37.123456,
          "longitude": -122.123456
        }
      },
      {
        "name": "cg-tm3",
        "coordinates": {
          "latitude": 38.123456,
          "longitude": -121.123456
        }
      },
      {
        "name": "cg-tm4",
        "coordinates": {
          "latitude": 37.123456,
          "longitude": -120.123456
        }
      },
      {
        "name": "cg-tm5",
        "coordinates": {
          "latitude": 38.123456,
          "longitude": -120.123456
        }
      },
      {
        "name": "cg-tm6",
        "coordinates": {
          "latitude": 36.123456,
          "longitude": -120.123456
        }
      },
      {
        "name": "cg-tm7",
        "coordinates": {
          "latitude": 36.123456,
          "longitude": -122.123456
        }
      },
      {
        "name": "cg-tm8",
        "coordinates": {
          "latitude": 36.123456,
          "longitude": -123.123456
        }
      },
      {
        "name": "cg-tm9",
        "coordinates": {
          "latitude": 39.123456,
          "longitude": -123.123456
        }
      }
    ],
    "profiles": [
      {
        "name": "pr-ec0",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec1",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec2",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec3",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec4",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec5",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec6",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec7",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec8",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      },
      {
        "name": "pr-ec9",
        "type": "EDGE",
        "parameters": {
          "health.connection.timeout": 2000,
          "health.polling.url": "http://${hostname}/_astats?application=&inf.name=${interface_name}",
          "health.threshold.availableBandwidthInKbps": ">1750000",
          "health.threshold.loadavg": 32,
          "health.threshold.queryTime": 1000,
          "history.count": 30
        }
      }
    ],
    "deliveryServices": [],
    "config": {
      "hack.ttl": 30,
      "health.event-count": 200,
      "health.polling.interval": 60000,
      "health.threadPool": 4,
      "health.timepad": 0,
      "heartbeat.polling.interval": 20000,
      "location": "/opt/traffic_monitor/conf",
      "peers.polling.interval": 10000,
      "tm.crConfig.polling.url": "https://${tmHostname}/CRConfig-Snapshots/${cdnName}/CRConfig.xml",
      "tm.dataServer.polling.url": "https://${tmHostname}/dataserver/orderby/id",
      "tm.healthParams.polling.url": "https://${tmHostname}/health/${cdnName}",
      "tm.polling.interval": 60000
    }
}
`
	obj := TrafficMonitorConfig2{}
	if err := json.Unmarshal([]byte(jstr), &obj); err != nil {
		panic(err)
	}
	return &obj
}
