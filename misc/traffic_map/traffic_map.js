var map;

var InfluxURL = "";

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

var GroupedLayers;

var LatLonStats = {};
var overlayMapsStats = {};
var StatsTtmsStateLayers = {};
var ZipcodeTtms = {};
var StateTtms = {};

var info; // info pane

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

function ajaxGetWithData(url, params, callback){
  var http = new XMLHttpRequest();
  http.open("GET", url, true);
  http.setRequestHeader("Content-type", "application/x-www-form-urlencoded");
  http.onreadystatechange = function() {//Call a function when the state changes.
    if(http.readyState == 4 && http.status == 200) {
      callback(http.responseText);
    }
  }
  http.send(params);
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

function createLegend() {
  var legend = L.control({position: 'bottomright'});

  var ColorOffset = 1.5 // needs to be yellower, green is stronger than yellow.

  legend.onAdd = function (map) {
    var div = L.DomUtil.create('div', 'info legend'),
    grades = [2.0, 1.75, 1.5, 1.0, 0.5, 0.0]
    labels = [];

    // loop through our density intervals and generate a label with a colored square for each interval
    for (var i = 0; i < grades.length-1; i++) {
      div.innerHTML += '<i style="background:' + getColor(normalizeTtmsRatio(grades[i+1])) + '"></i> ' + (i == 0 ? '+':'&nbsp&nbsp') + grades[i].toFixed(2) + '&ndash;' + grades[i+1].toFixed(2) + '<br>';
    }

    return div;
  };
  legend.addTo(map);
}

function ratioToPercentStr(ratio) {
  return ((1.0-ratio)*100).toString().substring(0, 2);
}

function createInfo() {
  info = L.control();

  info.onAdd = function (map) {
    this._div = L.DomUtil.create('div', 'info'); // create a div with a class "info"
    this.update();
    return this._div;
  };

  // method that we will use to update the control based on feature properties passed
  info.update = function (props) {
     this._div.innerHTML = '<h4>Customer Experience Ratio</h4>' +  (props ?
																																'<b>' + props.NAME + '</b><br />' + props.ttmsRatio.toFixed(2) + ''
																																: 'Hover over a state');
  };

  info.addTo(map);
}

function initMap(tileUrl) {
  map = new L.Map('map0');

  var osmAttrib='Map data © <a href="http://openstreetmap.org">OpenStreetMap</a> contributors'; // TODO fix? I'm hesitant to make this a parameter which can be omitted, encouraging OSM TOS violations.
  var osm = new L.TileLayer(tileUrl, {minZoom: 2, maxZoom: 16, attribution: osmAttrib});
  map.setView(new L.LatLng(39.73, -104.98),5);
  map.addLayer(osm);
  createLegend();
  createInfo();
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

    getRegions();
  })
}

// normalizeTtmsRatio takes a TTMS ratio as returned by calcTtmsRatio (0.0-2.0+) and returns a number between 0.0 and 1.0 where 0.0 is bad and 1.0 is good.
function normalizeTtmsRatio(d) {
  var oldD = d;
  var ColorBadnessDivisor = 0.5; // debug - color value is divided by this ratio, lower means less green and more red
  d = d * ColorBadnessDivisor;
  if(d > 2.0) {
    d = 2.0;
  }
 d = 1.0 - (d/2.0);
 if(d < 0.0001) {
    d = 0.0001;
  }
  // console.log("normalizeTtmsRatio got " + oldD + " returning " + d);
  return d;
}

// 0.0 < d < 1.0
function getColor(d) {
  if(d > 0.9999) {
    d = 0.9999;
  } else if(d < 0.0001) {
    d = 0.0001;
  }
  var hexColorNum = d*256;
  var hexStr = hexColorNum.toString(16);
  var hexStrNoDot = hexStr;
  if(hexStr.indexOf('.') != -1) {
    hexStrNoDot = hexStr.substring(0, hexStr.indexOf('.'));
  }
  var hexStrTwo = hexStrNoDot.substring(0, 2);
  var hexStrLengthened = hexStrTwo;
  if(hexStrLengthened.length == 1) {
    hexStrLengthened = "0" + hexStrLengthened;
  }

  var colorStr = '#'+hexStrLengthened+"cc00";
  // console.log('colorStr ' + d + ' is ' + colorStr + ' from ' + hexStr + '->' + hexStrNoDot + '->' + hexStrTwo + '->' + hexStrLengthened)
  return colorStr;
}

// function getColor(d) {
//   d = 0.7;
//   if(d > 0.9999) {
//     d = 0.9999;
//   } else if(d < 0.0001) {
//     d = 0.0001;
//   }
//   if (d == 0.5) {
//     return "#ffff00"
//   }
//   if (d < 0.5) {
//     var hexStr = (d * 510).toString(16) //generate a range from 0 to 255
//     var hexStrNoDot = hexStr;
//     if(hexStr.indexOf('.') != -1) {
//       hexStrNoDot = hexStr.substring(0, hexStr.indexOf('.'));
//     }

//     var hexStr2 = ((d * 102)+204).toString(16) //generate a range from 204 to 255
//     var hexStrNoDot2 = hexStr2;
//     if(hexStr2.indexOf('.') != -1) {
//       hexStrNoDot2 = hexStr2.substring(0, hexStr2.indexOf('.'));
//     }

//     //build the string
//     var str = "#" +
//       ("0" + hexStrNoDot).slice(-2) +
//       ("0" + hexStrNoDot2).slice(-2) +
//       "00";
//     return str;

//   } else {
//     var hexStr = (510-(d * 510)).toString(16) //generate a range from 255 to 0
//     var hexStrNoDot = hexStr;
//     if(hexStr.indexOf('.') != -1) {
//       hexStrNoDot = hexStr.substring(0, hexStr.indexOf('.'));
//     }

//     var str = "#ff" +
//       ("0" + hexStrNoDot).slice(-2) +
//       "00";
//     return str;
//   }
// }

function ttmsStyle(feature) {
    return {
        fillColor: getColor(normalizeTtmsRatio(feature.properties.ttmsRatio)),
        weight: 2,
        opacity: 1,
        color: 'white',
        dashArray: '3',
        fillOpacity: 0.7
    };
}

var AverageFragmentLengthMS = 2000;
// calcTtmsRatio returns a number between 0.0 and 2.0+ where 2.0+ is considered "perfect"
function calcTtmsRatio(ttms) {
  if(typeof ttms == "undefined" || ttms == "") {
      return 2.0 // TODO stop returning ideal ratios for missing data
  }
  if(ttms == 0.0) {
    ttms = 0.001;
  }
  return AverageFragmentLengthMS/ttms;
}

function getZipcodeTtms() {
  var series = LatLonStats.results[0].series;
  for(var seriesi = 0; seriesi < series.length; seriesi++) {
    var serie = series[seriesi];
    if(serie.name != "ttms_data") {
      continue;
    }
    var zipcode = serie.tags.postcode;
    var stateName = ZipToStateName[zipcode];
    if(typeof stateName == "undefined" || stateName == "") {
      continue;
    }

    var latestValue = -1;
    // TODO invert loop, break after find (last is latest)
    for(var valuesi = 0; valuesi < serie.values.length; valuesi++) {
      var value = serie.values[valuesi];
      if(value[1] == null) {
        continue;
      }
      latestValue = value[1];
    }
    ZipcodeTtms[zipcode] = latestValue;
  }
}

function getStateTtms() {
  var stateTtmses = {};
  for(var zipcode in ZipcodeTtms) {
    var ttms = ZipcodeTtms[zipcode];
    var stateName = ZipToStateName[zipcode];
    if(typeof stateTtmses[stateName] == "undefined") {
      stateTtmses[stateName] = [];
    }
    stateTtmses[stateName].push(ttms);
    // stateZipcodes[stateName] = stateZipcodes[stateName] + 1;
  }

  for(var state in stateTtmses) {
    var stateTtms = stateTtmses[state];
    var sum = 0;
    for( var ttmsi = 0; ttmsi < stateTtms.length; ttmsi++ ){
      sum += stateTtms[ttmsi];
    }
    var avg = sum/stateTtms.length;
    StateTtms[state] = avg;
  }
}

function highlightFeature(e) {
    var layer = e.target;

    // layer.setStyle({
    //     weight: 5,
    //     color: '#666',
    //     dashArray: '',
    //     fillOpacity: 0.7
    // });

    // if (!L.Browser.ie && !L.Browser.opera && !L.Browser.edge) {
    //     layer.bringToFront();
    // }

  info.update(layer.feature.properties);
}

function resetHighlight(e) {
  // geojson.resetStyle(e.target);

  info.update();
}

function onEachFeature(feature, layer) {
    layer.on({
        mouseover: highlightFeature,
        // mouseout: resetHighlight,
        // click: zoomToFeature
    });
}

function calcStateStats() {
  console.log("Calculating State Stats");
  overlayMapsStats["None"] = L.layerGroup();

  var ttmsLayerName = "Customer Experience Ratio";

  var lg = L.layerGroup();
  overlayMapsStats[ttmsLayerName] = lg

  // var myStyle = {
  //   "color": "#ff7800",
  //   "weight": 5,
  //   "opacity": 0.65
  // };

  for(var usstateI = 0; usstateI < USStatesGeoJSON.features.length; usstateI++) {
    var usState = USStatesGeoJSON.features[usstateI];
    var usStateName = usState.properties.NAME;

    usState.properties.ttmsRatio = calcTtmsRatio(StateTtms[usStateName]);

    if(typeof StateTtms[usStateName] == "undefined" || StateTtms[usStateName] == "") {
      console.log("State TTMS Ratio " + usStateName + " " + "NO DATA");
    }
    // console.log("State TTMS Ratio " + usStateName + " " + usState.properties.ttmsRatio + " color " + getColor(normalizeTtmsRatio(usState.properties.ttmsRatio)));

    var layer = L.geoJSON(usState, {style: ttmsStyle, onEachFeature: onEachFeature});
    layer.addTo(map); // immediately display "Customer Experience Ratio" map
    StatsTtmsStateLayers[usStateName] = layer;
    overlayMapsStats[ttmsLayerName].addLayer(layer);
  }

  var groupedOverlays = {
    // "CDNs": overlayMapsCdn,
    "Stats": overlayMapsStats,
    // "Delivery Services": overlayMapsDs,
  };
  var groupedLayersOptions = {
    exclusiveGroups: ["CDNs", "Stats", "Delivery Services"],
  };
  // TODO move to init, and call `GroupedLayers.addOverlay(layer, name, group)` here
  GroupedLayers = L.control.groupedLayers(null, groupedOverlays, groupedLayersOptions)
  GroupedLayers.addTo(map);

  // GroupedLayers.addOverlay(StatsTtmsStateLayers, "Customer Experience Ratio", "Stats")

  // var groupedOverlays = {
  //   "CDNs": overlayMapsCdn,
  //   "Delivery Services": overlayMapsDs,
  // };
  // var groupedLayersOptions = {
  //   exclusiveGroups: ["CDNs", "Delivery Services"],
  // };
  // // TODO move to init, and call `GroupedLayers.addOverlay(layer, name, group)` here
  // GroupedLayers = L.control.groupedLayers(null, groupedOverlays, groupedLayersOptions)
  // GroupedLayers.addTo(map);

  map.invalidateSize();
  console.log("Done");
}

function getLatlonStats() {
  console.log("Getting Latlon Stats ("+InfluxURL+")");
  var params = 'db=' + encodeURIComponent('latlon_stats') + "&q=" + encodeURIComponent('select mean(ttms) from ttms_data where time > now() - 60m group by postcode, time(30m)');
  ajax(InfluxURL+"/query?"+params, function(srvTxt) {
    LatLonStats = JSON.parse(srvTxt);
    // console.log("LatLonStats: " + srvTxt);
    if(typeof LatLonStats.results[0].series != "undefined") {
      getZipcodeTtms();
      getStateTtms();
      calcStateStats();
    } else {
      console.log("Influx returned no series!")
      console.log("Done (failed)")
    }
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
  })
}

function getZipStates() {
  console.log("Getting Zipcodes");
  ajax("/zip-to-state-name.json", function(srvTxt) {
    var raw = JSON.parse(srvTxt);
    ZipToStateName = JSON.parse(srvTxt);
    getLatlonStats();
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

function init(tileUrl, influxUrl) {
  // console.log("color 0.9 is " + getColor(0.9))
  // console.log("color 0.1 is " + getColor(0.1))
  InfluxURL = influxUrl;
  initMap(tileUrl);
  // getCDNs();
  getRegions();
}
