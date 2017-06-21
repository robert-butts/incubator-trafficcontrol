var map;

var cacheUnknownColor = '#000000';
var cacheAvailableColor = '#009900';
var cacheOfflineColor = '#FFBB00';
var cacheUnavailableColor = '#FF0000';

var iconSize = [25, 25];
var cgIcon = L.icon({
  iconUrl: 'cg-grey.png',
  iconSize:     iconSize,
  iconAnchor:   [0, 0],
  popupAnchor:  [10, 2]
});
var cgErrIcon = L.icon({
  iconUrl: 'cg-red.png',
  iconSize:     iconSize,
  iconAnchor:   [0, 0],
  popupAnchor:  [-3, -76]
});
var cgWarnIcon = L.icon({
  iconUrl: 'cg-orange.png',
  iconSize:     iconSize,
  iconAnchor:   [0, 0],
  popupAnchor:  [-3, -76]
});
var servers = {};
var cachegroups = {};
var cachegroupMarkers = {};
var cacheCachegroups = {};
var cachegroupCaches = {};
var cachePopupElems = {};

var cdns = {};
var cdnServerLayerGroups = {};
var overlayMapsCdn = {};
var overlayMapsDs = {};

var deliveryServices = {};
var deliveryServiceMarkers = {}; // deliveryServiceMarkers[deliveryServiceName][cachegroup]marker
var deliveryServiceLayerGroups = {};

var crconfigs = {};
var deliveryServiceServers = {};
var deliveryServiceCachegroupServers = {};

var USStatesGeoJSON = {};
var CachegroupUSStates = {};

var ZipToStateName = {};

function ajax(url, callback){
  var xmlhttp = new XMLHttpRequest();
  xmlhttp.onreadystatechange = function(){
    if (xmlhttp.readyState == 4 && xmlhttp.status == 200){
      callback(xmlhttp.responseText);
    }
  }
  xmlhttp.open("GET", url, true);
  xmlhttp.send();
}

// pointInGeometry returns whether the given [lat, lon] is within the given GeoJSON Geometry Polygon or MultiPolygon. Note this does not support inner rings ("holes") yet.
function GeoJSONLatLonInFeature(lonlat, feature) {
  if(feature.geometry.type == "Polygon") {
    return pointInPolygon(lonlat, feature.geometry.coordinates[0]);
  }
  if(feature.geometry.type == "MultiPolygon") {
    for(var i = 0; i < feature.geometry.coordinates.length; i++) {
      if(pointInPolygon(lonlat, feature.geometry.coordinates[i][0])) {
        return true;
      }
    }
    return false;
  }
  return false;
}

// pointInPolygon from https://github.com/substack/point-in-polygon licensed MIT
// Note point is lon-lat not lat-lon, because GeoJSON is lon-lat
function pointInPolygon(point, vs) {
  // ray-casting algorithm based on
  // http://www.ecse.rpi.edu/Homepages/wrf/Research/Short_Notes/pnpoly.html

  var x = point[0], y = point[1];

  var inside = false;
  for (var i = 0, j = vs.length - 1; i < vs.length; j = i++) {
    var xi = vs[i][0], yi = vs[i][1];
    var xj = vs[j][0], yj = vs[j][1];

    var intersect = ((yi > y) != (yj > y))
        && (x < (xj - xi) * (y - yi) / (yj - yi) + xi);
    if (intersect) inside = !inside;
  }

  return inside;
}


function initMap(tileUrl) {
  map = new L.Map('map0');

  var osmAttrib='Map data © <a href="http://openstreetmap.org">OpenStreetMap</a> contributors'; // TODO fix? I'm hesitant to make this a parameter which can be omitted, encouraging OSM TOS violations.
  var osm = new L.TileLayer(tileUrl, {minZoom: 2, maxZoom: 16, attribution: osmAttrib});
  map.setView(new L.LatLng(39.73, -104.98),5);
  map.addLayer(osm);
}

function getCachegroupMarkerPopup(cg) {
  var div = document.createElement("div");

  var b = document.createElement("b");
  div.appendChild(b);

  var txt = document.createTextNode(cg.name);
  b.appendChild(txt);

  var br = document.createElement("br");
  div.appendChild(br);

  return div
}

function addCache(cachegroupMarkerPopupContent, cacheName) {
  var span = document.createElement("span");
  span.style.color = cacheUnknownColor;
  span.style.margin = "10px";
  var txt = document.createTextNode(cacheName);
  span.appendChild(txt);
  cachegroupMarkerPopupContent.appendChild(span);

  cachePopupElems[cacheName] = span;
  return cachegroupMarkerPopupContent;
}

function getStates() {
  console.log("Getting Server State");
  ajax("/publish/CrStates", function(srvTxt) {
    var rawStates = JSON.parse(srvTxt);
    var cacheStates = rawStates["caches"];
    for(var cacheName in cacheStates) {
      if (!cacheStates.hasOwnProperty(cacheName)) {
        continue; // skip prototype properties
      }

      var cacheElem = cachePopupElems[cacheName];
      if(typeof cacheElem == "undefined") {
        // console.log("ERROR: cache " + cacheName + " has no element!"); // DEBUG
        continue
      }
      var available = cacheStates[cacheName].isAvailable;
      if(available) {
        cacheElem.style.color = cacheAvailableColor;
        cacheElem.style.fontWeight = 'normal';
      } else {
        /* console.log("cache " + cacheName + " is " + available); */
        cacheElem.style.color = cacheUnavailableColor;
        cacheElem.style.fontWeight = 'bold';
      }
    }
    getCRConfigs(cdns);
  })
}

function getCRConfigs(cdns) {
  if(cdns.length == 0) {
    getDeliveryServicesState();
    return
  }
  var cdn = cdns[0];
  if(cdn.name == 'ALL') {
    getCRConfigs(cdns.slice(1));
    return;
  }
  console.log("Getting CDN Config " + cdn.name);
  ajax('/CRConfig-Snapshots/' + cdn.name + '/CRConfig.json', function(srvTxt) {
    var crconfig = JSON.parse(srvTxt);
    crconfigs[cdn.name] = crconfig;
    addDeliveryServiceServers(crconfig);
    getCRConfigs(cdns.slice(1));
  })
}

function addDeliveryServiceServers(crconfig) {
  var servers = crconfig["contentServers"];
  for(var server in servers) {
    if (!servers.hasOwnProperty(server)) {
      continue; // skip prototype properties
    }
    deliveryServices = servers[server].deliveryServices;
		cachegroup = servers[server].cacheGroup
    for(var deliveryService in deliveryServices) {
      if(!deliveryServices.hasOwnProperty(deliveryService)) {
        continue; // skip prototype properties
      }
			if(!deliveryServiceServers.hasOwnProperty(deliveryService)) {
				deliveryServiceServers[deliveryService] = [];
			}
      deliveryServiceServers[deliveryService].push(server);

      if(!deliveryServiceCachegroupServers.hasOwnProperty(deliveryService)) {
        deliveryServiceCachegroupServers[deliveryService] = {};
      }
      if(!deliveryServiceCachegroupServers[deliveryService].hasOwnProperty(cachegroup)) {
        deliveryServiceCachegroupServers[deliveryService][cachegroup] = [];
      }
      deliveryServiceCachegroupServers[deliveryService][cachegroup].push(server);
    }
  }
}

function getDeliveryServicesState() {
  console.log("Getting Deliveryservice State");
  ajax("/publish/DsStats", function(srvTxt) {
    var raw = JSON.parse(srvTxt);
    deliveryServices = raw["deliveryService"];

    var lgDsNone = L.layerGroup();
    deliveryServiceLayerGroups["None"] = lgDsNone;
    overlayMapsDs["None"] = lgDsNone;

    for(var deliveryService in deliveryServices) {
      deliveryServiceMarkers[deliveryService] = {};

      var markers = [];
      for(var j = 0; j < cachegroups.length; j++) {
        var cg = cachegroups[j];

        if(deliveryServiceCachegroupServers.hasOwnProperty(deliveryService) && deliveryServiceCachegroupServers[deliveryService].hasOwnProperty(cg.name)) {
          var cgMarker = L.AwesomeMarkers.icon({
            icon: 'coffee',
            markerColor: 'blue',
            html: deliveryServiceCachegroupServers[deliveryService][cg.name].length
          });
          var marker = L.marker([cg.latitude, cg.longitude], {icon: cgMarker});
          var popup = marker.bindPopup(getCachegroupMarkerPopup(cg));
          deliveryServiceMarkers[deliveryService][cg.name] = marker;
          markers.push(marker)
        }
      }
      var layerGroup = L.layerGroup(markers);
      overlayMapsDs[deliveryService] = layerGroup
      deliveryServiceLayerGroups[deliveryService] = layerGroup;
    }

    var groupedOverlays = {
      "CDNs": overlayMapsCdn,
      "Delivery Services": overlayMapsDs,
    };
    var groupedLayersOptions = {
      exclusiveGroups: ["CDNs", "Delivery Services"],
    };
    L.control.groupedLayers(null, groupedOverlays, groupedLayersOptions).addTo(map);

    getRegions();
  })
}

function calcCachegroupUSStates() {
  for(var cachegroupI = 0; cachegroupI < cachegroups.length; cachegroupI++) {
    var cachegroup = cachegroups[cachegroupI];
    var cachegroupLonLat = [cachegroup.longitude, cachegroup.latitude];
    for(var usstateI = 0; usstateI < USStatesGeoJSON.features.length; usstateI++) {
      var usState = USStatesGeoJSON.features[usstateI];
      if(GeoJSONLatLonInFeature(cachegroupLonLat, usState)) {
        CachegroupUSStates[cachegroup.name] = usState.properties.NAME;
      }
    }
  }
}

function getRegions() {
  console.log("Getting Regions");
  ajax("/us-states-geojson.min.json", function(srvTxt) {
    USStatesGeoJSON = JSON.parse(srvTxt);
    calcCachegroupUSStates();
    getZipStates();
    console.log("Done");
  })
}

function getZipStates() {
  console.log("Getting Zipcodes");
  ajax("/zip-to-state-name.json", function(srvTxt) {
    var raw = JSON.parse(srvTxt);
    ZipToStateName = JSON.parse(srvTxt);
    console.log("Done");
  })
}

// function hostnameFromFqdn(fqdn) {
//   var dotPos = fqdn.indexOf(".");
//   if(dotPos == -1) {
//     return fqdn;
//   }
//   var hostname = fqdn.substring(0, dotPos);
//   return hostname;
// }

function addServerToMarker(server, cdnName) {
  var cacheName = server.hostName;
  var cgName = server.cachegroup;
  var marker = cachegroupMarkers[cdnName][cgName];
  if(typeof marker == "undefined") {
    console.log("ERROR no cachegroup for " + cgName);
    return;
  }
  var popup = marker.getPopup();
  var popupContent = popup.getContent();
  popupContent = addCache(popupContent, cacheName);
  popup.setContent(popupContent); // TODO necessary?
  popup.update(); // TODO update once per popup? Necessary?
}

function getServers() {
  console.log("Getting Servers");
  ajax("/api/1.2/servers.json", function(srvTxt) {
    var rawServers = JSON.parse(srvTxt);
    servers = rawServers["response"];
    for(var i = 0; i < servers.length; i++) {
      var s = servers[i];
      var cacheName = s.hostName;
      var cgName = s.cachegroup;
      var cdnName = s.cdnName;

      addServerToMarker(s, cdnName);
      addServerToMarker(s, "ALL");

      cacheCachegroups[cacheName] = cgName;
      if(typeof cachegroupCaches[cgName] == "undefined") {
        cachegroupCaches[cgName] = [];
      }
      cachegroupCaches[cgName].push(cgName);
    }
    getStates()
  })
}

function getCachegroups() {
  console.log("Getting Cachegroups");
  ajax("/api/1.2/cachegroups.json", function(cgTxt) {
    var rawCachegroups = JSON.parse(cgTxt);
    cachegroups = rawCachegroups["response"];
    for(var i = 0; i < cdns.length; i++) {
      var cdn = cdns[i];
      // console.log("cachegroupMarkers cdn " + cdn.name); // DEBUG
      cachegroupMarkers[cdn.name] = {};
      for(var j = 0; j < cachegroups.length; j++) {
        var cg = cachegroups[j];
        var marker = L.marker([cg.latitude, cg.longitude], {icon: cgIcon});
        var popup = marker.bindPopup(getCachegroupMarkerPopup(cg));
        cachegroupMarkers[cdn.name][cg.name] = marker;
        cdnServerLayerGroups[cdn.name].addLayer(marker);
      }
    }
    getServers(); // TODO concurrently request with cachegroups
  })
}

function getCDNs() {
  console.log("Getting CDNs");
  ajax("/api/1.2/cdns.json", function(txt) {
    var raw = JSON.parse(txt);
    cdns = raw["response"];

    var lgCdnNone = L.layerGroup();
    cdnServerLayerGroups["None"] = lgCdnNone;
    overlayMapsCdn["None"] = lgCdnNone;

    for(var i = 0; i < cdns.length; i++) {
      var cdn = cdns[i];
			var lg = L.layerGroup();
			cdnServerLayerGroups[cdn.name] = lg;
			overlayMapsCdn[cdn.name] = lg
		}
		getCachegroups();
	})
}

function init(tileUrl) {
  initMap(tileUrl);
	getCDNs();
}
