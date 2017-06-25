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

var USCountiesGeoJSON = {};
var CachegroupUSCounties = {};

var ZipToStateName = {};
var ZipToStateCounty = {}; // value is state-space-county

var GroupedLayers;

var LatLonStats = {};
var overlayMapsStats = {};
// var StatsTtmsStateLayers = {};
var DeliveryserviceZipcodeTtms = {};
var DeliveryserviceStateTtms = {};
var DeliveryserviceCountyTtms = {};

var cdnCachegroupServers = {};

var info; // info pane

var ColorBadnessDivisor = 1.0;

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

// TODO abstract duplicate code
function reloadLegendStyle() {
  var span = document.getElementById("legendSpan")
  grades = [2.0, 1.75, 1.5, 1.0, 0.5, 0.0]
  labels = [];

  // loop through our density intervals and generate a label with a colored square for each interval
  span.innerHTML = "";
  for (var i = 0; i < grades.length-1; i++) {
    span.innerHTML += '<i style="background:' + getColor(normalizeTtmsRatio(grades[i+1])) + '"></i> ' + (i == 0 ? '+':'&nbsp&nbsp') + grades[i].toFixed(2) + '&ndash;' + grades[i+1].toFixed(2) + '<br>';
  }
}

function createLegend() {
  var legend = L.control({position: 'bottomright'});

  var ColorOffset = 1.5 // needs to be yellower, green is stronger than yellow.

  legend.onAdd = function (map) {
    var div = L.DomUtil.create('div', 'info legend');
    div.id = "legendDiv";
    grades = [2.0, 1.75, 1.5, 1.0, 0.5, 0.0]
    labels = [];

    var legendSpan = document.createElement('span');
    legendSpan.id = "legendSpan";
    // loop through our density intervals and generate a label with a colored square for each interval
    for (var i = 0; i < grades.length-1; i++) {
      legendSpan.innerHTML += '<i style="background:' + getColor(normalizeTtmsRatio(grades[i+1])) + '"></i> ' + (i == 0 ? '+':'&nbsp&nbsp') + grades[i].toFixed(2) + '&ndash;' + grades[i+1].toFixed(2) + '<br>';
    }
    div.appendChild(legendSpan);

    // div.innerHTML += '<h4>Intensity</h4>';
    var slider = document.createElement("INPUT");
    slider.className += " slider";
    slider.setAttribute("type", "range");
    slider.setAttribute("min", "1");
    slider.setAttribute("max", "99");
    slider.setAttribute("step", "1");
    slider.value = 1;
    slider.oninput = function() {
      ColorBadnessDivisor = (100 - slider.value) / 100;
      reloadStatsStyles();
      map.invalidateSize();
    }
    div.appendChild(slider);

    return div;
  };

  legend.addTo(map);

  // Disable dragging when user's cursor enters the element
  legend.getContainer().addEventListener('mouseover', function () {
    map.dragging.disable();
  });

  // Re-enable dragging when user's cursor leaves the element
  legend.getContainer().addEventListener('mouseout', function () {
    map.dragging.enable();
  });
}

function ratioToPercentStr(ratio) {
  return ((1.0-ratio)*100).toString().substring(0, 2);
}

function getGeoJSONPropertiesDisplayName(properties) {
  if(typeof properties.COUNTY == "undefined") {
    return properties.NAME;
  }
  return toTitleCase(properties.NAME) + ", " + properties.STATE;

}

function startsWith(s, prefix) {
  return s.substring(0, prefix.length) === prefix;
}
function endsWith(s, suffix) {
 return s.indexOf(suffix, s.length - suffix.length) !== -1;
}
function contains(s, val) {
 return s.indexOf(val) != -1;
}

function showCachegroup(cg) {
  if(cg.typeName != "EDGE_LOC") {
    return false;
  }
  if(!startsWith(cg.name, "us-")) {
    return false;
  }
  if(startsWith(cg.name, "us-bb-")) {
    return false;
  }
  if(endsWith(cg.name, "lab")) {
    return false;
  }
  if(contains(cg.name, "cox")) {
    return false;
  }
  return true;
}

var CurrentRegionType = "";

function createInfo() {
  info = L.control();

  info.onAdd = function (map) {
    this._div = L.DomUtil.create('div', 'info'); // create a div with a class "info"
    this.update();
    return this._div;
  };

  // method that we will use to update the control based on feature properties passed
  info.update = function (props) {
    if(props) {
      if(props.ttmsRatio > 0.0) {
        this._div.innerHTML = '<h4>Customer Experience Ratio</h4>' + '<b>' + getGeoJSONPropertiesDisplayName(props) + '</b><br />' + props.ttmsRatio.toFixed(2) + '';
      } else {
        this._div.innerHTML = '<h4>Customer Experience Ratio</h4>' + '<b>' + getGeoJSONPropertiesDisplayName(props) + '</b><br />' + '<i>no recent customers</i>' + '';
      }
    } else {
      this._div.innerHTML = '<h4>Customer Experience Ratio</h4>';
      if(typeof CurrentRegionType != "undefined" && CurrentRegionType != "") {
        this._div.innerHTML += 'Hover over a ' + CurrentRegionType;
      }
    }
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
  map.on('layeradd', onLayerAdd);
  map.on('overlayadd', onOverlayAdd);
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
  topbar.innerHTML = "Loading CDN Server State";
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
  topbar.innerHTML = "Loading CDN Config";
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

      if(!deliveryServicecachegroupServers.hasOwnProperty(deliveryService)) {
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
  topbar.innerHTML = "Loading Deliveryservice State";
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
  if(d == -1) {
    return d
  }
  var oldD = d;

  d = d * ColorBadnessDivisor;
  var badnessChange = d;
  if(d > 2.0) {
    d = 2.0;
  }
 d = 1.0 - (d/2.0);
 if(d < 0.0001) {
    d = 0.0001;
  }
  // console.log("normalizeTtmsRatio got " + oldD + " adjusted to " + badnessChange + " returning " + d);
  return d;
}

// 0.0 < d < 1.0
function getColor(d) {
  if(d == -1) {
    return "#9999aa" // missing data => blue
  }

  if(d > 0.9999) {
    d = 0.9999;
  } else if(d < 0.0001) {
    d = 0.0001;
  }
  if (d == 0.5) {
    return "#ffff00"
  }
  if (d < 0.5) {
    var hexStr = (d * 510).toString(16) //generate a range from 0 to 255
    var hexStrNoDot = hexStr;
    if(hexStr.indexOf('.') != -1) {
      hexStrNoDot = hexStr.substring(0, hexStr.indexOf('.'));
    }

    var hexStr2 = ((d * 102)+204).toString(16) //generate a range from 204 to 255
    var hexStrNoDot2 = hexStr2;
    if(hexStr2.indexOf('.') != -1) {
      hexStrNoDot2 = hexStr2.substring(0, hexStr2.indexOf('.'));
    }

    //build the string
    var str = "#" +
      ("0" + hexStrNoDot).slice(-2) +
      ("0" + hexStrNoDot2).slice(-2) +
      "00";
    return str;

  } else {
    var hexStr = (510-(d * 510)).toString(16) //generate a range from 255 to 0
    var hexStrNoDot = hexStr;
    if(hexStr.indexOf('.') != -1) {
      hexStrNoDot = hexStr.substring(0, hexStr.indexOf('.'));
    }

    var str = "#ff" +
      ("0" + hexStrNoDot).slice(-2) +
      "00";
    return str;
  }
}

function ttmsStyle(feature) {
    return {
        fillColor: getColor(normalizeTtmsRatio(feature.properties.ttmsRatio)),
        weight: 1,
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
      return -1 // TODO stop returning ideal ratios for missing data
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
    var deliveryservice = serie.tags.deliveryservice;
    var stateName = ZipToStateName[zipcode];
    if(typeof stateName == "undefined" || stateName == "") {
			console.log("ZipToStateName[" + zipcode + "] undefined!");
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

    if(typeof DeliveryserviceZipcodeTtms[deliveryservice] == "undefined") {
      DeliveryserviceZipcodeTtms[deliveryservice] = {};
    }
    DeliveryserviceZipcodeTtms[deliveryservice][zipcode] = latestValue;
  }
}

function getStateAndCountyTtms() {
  var deliveryserviceStateTtmses = {};
  var deliveryserviceCountyTtmses = {};

  for(var deliveryservice in DeliveryserviceZipcodeTtms) {
    var zipcodeTtms = DeliveryserviceZipcodeTtms[deliveryservice];
    if(typeof deliveryserviceStateTtmses[deliveryservice] == "undefined") {
      deliveryserviceStateTtmses[deliveryservice] = {};
    }
    if(typeof deliveryserviceCountyTtmses[deliveryservice] == "undefined") {
      deliveryserviceCountyTtmses[deliveryservice] = {};
    }

    for(var zipcode in zipcodeTtms) {
      var ttms = zipcodeTtms[zipcode];
      var stateName = ZipToStateName[zipcode];
      var countyName = ZipToStateCounty[zipcode];
      if(typeof deliveryserviceStateTtmses[deliveryservice][stateName] == "undefined") {
        deliveryserviceStateTtmses[deliveryservice][stateName] = [];
      }
      deliveryserviceStateTtmses[deliveryservice][stateName].push(ttms);

      if(typeof deliveryserviceCountyTtmses[deliveryservice][countyName] == "undefined") {
        deliveryserviceCountyTtmses[deliveryservice][countyName] = [];
      }
      deliveryserviceCountyTtmses[deliveryservice][countyName].push(ttms);
    }

    for(var deliveryservice in deliveryserviceStateTtmses) {
      if(typeof DeliveryserviceStateTtms[deliveryservice] == "undefined") {
        DeliveryserviceStateTtms[deliveryservice] = {};
      }
      for(var state in deliveryserviceStateTtmses[deliveryservice]) {
        var stateTtms = deliveryserviceStateTtmses[deliveryservice][state];
        var sum = 0;
        for( var ttmsi = 0; ttmsi < stateTtms.length; ttmsi++ ){
          sum += stateTtms[ttmsi];
        }
        var avg = sum/stateTtms.length;
        DeliveryserviceStateTtms[deliveryservice][state] = avg;
      }
    }

    for(var deliveryservice in deliveryserviceCountyTtmses) {
      if(typeof DeliveryserviceCountyTtms[deliveryservice] == "undefined") {
        DeliveryserviceCountyTtms[deliveryservice] = {};
      }
      for(var county in deliveryserviceCountyTtmses[deliveryservice]) {
        var countyTtms = deliveryserviceCountyTtmses[deliveryservice][county];
        var sum = 0;
        for( var ttmsi = 0; ttmsi < countyTtms.length; ttmsi++ ){
          sum += countyTtms[ttmsi];
        }
        var avg = sum/countyTtms.length;
        DeliveryserviceCountyTtms[deliveryservice][county] = avg;
      }
    }
  }
}

// function getStateTtms() {
//   var deliveryserviceStateTtmses = {};
//   for(var deliveryservice in DeliveryserviceZipcodeTtms) {
//     var zipcodeTtms = DeliveryserviceZipcodeTtms[deliveryservice];
//     if(typeof deliveryserviceStateTtmses[deliveryservice] == "undefined") {
//       deliveryserviceStateTtmses[deliveryservice] = {};
//     }
//     for(var zipcode in zipcodeTtms) {
//       var ttms = DeliveryserviceZipcodeTtms[deliveryservice][zipcode];
//       var stateName = ZipToStateName[zipcode];
//       if(typeof deliveryserviceStateTtmses[deliveryservice][stateName] == "undefined") {
//         deliveryserviceStateTtmses[deliveryservice][stateName] = [];
//       }
//       deliveryserviceStateTtmses[deliveryservice][stateName].push(ttms);
//     }

//     for(var deliveryservice in deliveryserviceStateTtmses) {
//       if(typeof DeliveryserviceStateTtms[deliveryservice] == "undefined") {
//         DeliveryserviceStateTtms[deliveryservice] = {};
//       }
//       for(var state in deliveryserviceStateTtmses[deliveryservice]) {
//         var stateTtms = deliveryserviceStateTtmses[deliveryservice][state];

//         var sum = 0;
//         for( var ttmsi = 0; ttmsi < stateTtms.length; ttmsi++ ){
//           sum += stateTtms[ttmsi];
//         }
//         var avg = sum/stateTtms.length;
//         DeliveryserviceStateTtms[deliveryservice][state] = avg;
//       }
//     }
//   }
// }

var CurrentLayer;
function onLayerAdd(e){
  // console.log("Setting current layer");
  CurrentLayer = e.target;
}

function onOverlayAdd(e){
  if(typeof e.layer.regionType != "undefined") {
    CurrentRegionType = e.layer.regionType;
    if(typeof info != "undefined") {
      info.update();
    }
  }
}


function highlightFeature(e) {
  var layer = e.target;

  // if(CurrentLayer != layer) {
  //   console.log("highlightFeature returning");
  //   return;
  // }
  // console.log("highlightFeature continuing");
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
        mouseout: resetHighlight,
        // click: zoomToFeature
    });
}

function reloadStatsStyles() {
  reloadLegendStyle();
  for(var overlayLayerName in overlayMapsStats) {
    var overlayLayer = overlayMapsStats[overlayLayerName];
    overlayLayer.eachLayer(function (layer) {
      layer.setStyle(ttmsStyle);
    });
  }
}

function calcStateStats() {
  topbar.innerHTML = "Calculating US State Stats";
  var lgStatsNone = L.layerGroup();
  lgStatsNone.regionType = "";
  overlayMapsStats["None"] = lgStatsNone

  var initialDeliveryServiceType =  "VOD"
  var initialLayer;

  for(var deliveryservice in DeliveryserviceStateTtms) {
    var dsType = "";
    if(deliveryservice == "col-jitp2") {
      dsType = "VOD"
    } else if(deliveryservice == "linear-nat-pil") {
      dsType = "Live"
    } else if(deliveryservice == "cdvr") {
      dsType = "DVR"
    } else if(deliveryservice == "cdvr-rio") {
      dsType = "DVR Rio"
    } else {
      console.log("Deliveryservice " + deliveryservice + " UNKNOWN - SKIPPING");
      continue;
    }

    var stateTtmsLayerName = dsType + " Customer Experience Ratio by State";
    var lg = L.layerGroup();
    lg.regionType = "state";

    overlayMapsStats[stateTtmsLayerName] = lg
    var stateTtms = DeliveryserviceStateTtms[deliveryservice];
    for(var usstateI = 0; usstateI < USStatesGeoJSON.features.length; usstateI++) {
      var usState = USStatesGeoJSON.features[usstateI];

      // copy, so states on different delivery services don't get the same properties (like ttms)
      // TODO more efficient copy?
      usState = JSON.parse(JSON.stringify(usState));

      var usStateName = usState.properties.NAME;

      usState.properties.ttmsRatio = calcTtmsRatio(stateTtms[usStateName]);
      if (usState.properties.ttmsRatio == -1) {
        continue; // don't draw regions with no data - comment this to draw dataless counties as grey, giving the illusion of a complete map overlay
      }

      var layer = L.geoJSON(usState, {style: ttmsStyle, onEachFeature: onEachFeature});
      overlayMapsStats[stateTtmsLayerName].addLayer(layer);
    }

    var countyTtmsLayerName = dsType + " Customer Experience Ratio by County";
    var lg = L.layerGroup();
    lg.regionType = "county";

    overlayMapsStats[countyTtmsLayerName] = lg
    if(dsType == initialDeliveryServiceType) {
      initialLayer = lg;
    }

    var countyTtms = DeliveryserviceCountyTtms[deliveryservice];
    for(var countyI = 0; countyI < USCountiesGeoJSON.features.length; countyI++) {
      var county = USCountiesGeoJSON.features[countyI];

      // copy, so geojson on different delivery services don't get the same properties (like ttms)
      // TODO more efficient copy?
      county = JSON.parse(JSON.stringify(county));

      var countyName = countyKeyNameGeoJSON(county);

      county.properties.ttmsRatio = calcTtmsRatio(countyTtms[countyName]);
      if (county.properties.ttmsRatio == -1) {
        continue; // don't draw regions with no data - comment this to draw dataless counties as grey, giving the illusion of a complete map overlay
      }

      var layer = L.geoJSON(county, {style: ttmsStyle, onEachFeature: onEachFeature});
      overlayMapsStats[countyTtmsLayerName].addLayer(layer);
    }
  }

  var groupedOverlays = {
    "CDNs": overlayMapsCdn,
    "Stats": overlayMapsStats,
    // "Delivery Services": overlayMapsDs,
  };
  var groupedLayersOptions = {
    exclusiveGroups: ["CDNs", "Stats", "Delivery Services"],
    collapsed: false,
    position: 'bottomleft'
  };
  // TODO move to init, and call `GroupedLayers.addOverlay(layer, name, group)` here
  GroupedLayers = L.control.groupedLayers(null, groupedOverlays, groupedLayersOptions)
  GroupedLayers.addTo(map);
  if(typeof initialLayer != "undefined") {
    initialLayer.addTo(map);
  }
  if(typeof overlayMapsCdn["ALL"] != "undefined") {
    overlayMapsCdn["ALL"].addTo(map);
  }
  createInfo();

  map.invalidateSize();
  toggleTop.checked = false;
}

function getLatlonStats() {
  topbar.innerHTML = "Loading Location Stats";
  // var params = 'db=' + encodeURIComponent('latlon_stats') + "&q=" + encodeURIComponent('select mean(ttms) from ttms_data where time > now() - 24h group by postcode, deliveryservice, time(24h)');
  // ajax(InfluxURL+"/query?"+params, function(srvTxt) {
  ajax('/query', function(srvTxt) {
    LatLonStats = JSON.parse(srvTxt);
    if(typeof LatLonStats.results[0].series != "undefined") {
      getZipcodeTtms();
      getStateAndCountyTtms();
      calcStateStats();
    } else {
      console.log("Influx returned no series!")
      topbar.innerHTML = "Failed To Load Location Stats (try refreshing the page)";
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


// from https://stackoverflow.com/questions/19974619/capitalize-first-letter-lowercase-the-rest-with-exceptions
function toTitleCase(str){
    return str.replace(/\w\S*/g, function(txt){return txt.charAt(0).toUpperCase() + txt.substr(1).toLowerCase();});
}
function countyKeyNameGeoJSON(county) {
  return county.properties.STATE + ", " + toTitleCase(county.properties.NAME); // TODO fix JSON to be title-cased
}

function countyKeyNameStrs(state, county) {
	return state + ", " + toTitleCase(county);
}

function calcCachegroupUSCounties() {
  for(var cachegroupI = 0; cachegroupI < cachegroups.length; cachegroupI++) {
    var cachegroup = cachegroups[cachegroupI];
    var cachegroupLonLat = [cachegroup.longitude, cachegroup.latitude];
    for(var countyI = 0; countyI < USCountiesGeoJSON.features.length; countyI++) {
      var county = USCountiesGeoJSON.features[countyI];
      if(GeoJSONLatLonInFeature(cachegroupLonLat, county)) {
        CachegroupUSCounties[cachegroup.name] = countyKeyNameGeoJSON(county);
      }
    }
  }
}


// TODO rename
function getRegions() {
  topbar.innerHTML = "Loading US State Data";
  ajax("/us-states-geojson.min.json", function(srvTxt) {
    topbar.innerHTML = "Parsing US State Data";
    USStatesGeoJSON = JSON.parse(srvTxt);
    calcCachegroupUSStates();
    getCounties();
  })
}

function getCounties() {
  topbar.innerHTML = "Loading US County Data";
  ajax("/us-counties-geojson.min.json", function(srvTxt) {
    USCountiesGeoJSON = JSON.parse(srvTxt);
    calcCachegroupUSCounties();
    getZipStates();
  })
}


function getZipStates() {
  topbar.innerHTML = "Loading Zipcode Data";
  ajax("/us-state-county-zips.min.json", function(srvTxt) {
    var raw = JSON.parse(srvTxt);
    zips = raw["result"];
    for(var zipi = 0; zipi < zips.length; zipi++) {
      var zip = zips[zipi];
      ZipToStateName[zip.Zipcode] = zip.State;
      ZipToStateCounty[zip.Zipcode] = countyKeyNameStrs(zip.State, zip.County);
    }
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

function addCachegroupLayers() {
  for(var i = 0; i < cdns.length; i++) {
    var cdn = cdns[i];
    cachegroupMarkers[cdn.name] = {};
    for(var j = 0; j < cachegroups.length; j++) {
      var cg = cachegroups[j];

      if(typeof cdnCachegroupServers[cdn.name] == "undefined") {
        continue; // if it's not in cachegroupServers, there are no servers in this cachegroup
      }
      if(typeof cdnCachegroupServers[cdn.name][cg.name] == "undefined") {
        continue; // if it's not in cachegroupServers, there are no servers in this cachegroup
      }

      var cgMarker = L.AwesomeMarkers.icon({
        icon: 'coffee',
        markerColor: 'blue',
        html: cdnCachegroupServers[cdn.name][cg.name].length
      });
      var marker = L.marker([cg.latitude, cg.longitude], {icon: cgMarker});
      var popup = marker.bindPopup(getCachegroupMarkerPopup(cg));
      cachegroupMarkers[cdn.name][cg.name] = marker;
      cdnServerLayerGroups[cdn.name].addLayer(marker);
      overlayMapsCdn[cdn.name].addLayer(marker);
    }
  }
}

function getServers() {
  topbar.innerHTML = "Loading CDN Server Data";
  ajax("/api/1.2/servers.json", function(srvTxt) {
    var rawServers = JSON.parse(srvTxt);
    servers = rawServers["response"];

    if(!cdnCachegroupServers.hasOwnProperty("ALL")) {
        cdnCachegroupServers["ALL"] = {};
    }

    for(var i = 0; i < servers.length; i++) {
      var s = servers[i];
      var cacheName = s.hostName;
      var cgName = s.cachegroup;
      var cdnName = s.cdnName;

      if(!cdnCachegroupServers.hasOwnProperty(cdnName)) {
        cdnCachegroupServers[cdnName] = {};
      }

      if(!cdnCachegroupServers[cdnName].hasOwnProperty(cgName)) {
        cdnCachegroupServers[cdnName][cgName] = [];
      }
      cdnCachegroupServers[cdnName][cgName].push(cacheName);

      if(!cdnCachegroupServers["ALL"].hasOwnProperty(cgName)) {
        cdnCachegroupServers["ALL"][cgName] = [];
      }
      cdnCachegroupServers["ALL"][cgName].push(cacheName);

      // addServerToMarker(s, cdnName);
      // addServerToMarker(s, "ALL");

      cacheCachegroups[cacheName] = cgName;
      if(typeof cachegroupCaches[cgName] == "undefined") {
        cachegroupCaches[cgName] = [];
      }
      cachegroupCaches[cgName].push(cgName);
    }
    addCachegroupLayers();
    getRegions();
    // getStates()
  })
}

function getCachegroups() {
  topbar.innerHTML = "Loading Cachegroups";
  ajax("/api/1.2/cachegroups.json", function(cgTxt) {
    var rawCachegroups = JSON.parse(cgTxt);
    cachegroups = [];
    for(var cachegroupI = 0; cachegroupI < rawCachegroups["response"].length; cachegroupI++) {
      var cachegroup = rawCachegroups["response"][cachegroupI];
      if(showCachegroup(cachegroup)) {
        cachegroups.push(cachegroup);
      }
    }
    getServers(); // TODO concurrently request with cachegroups
  })
}

function getCDNs() {
  topbar.innerHTML = "Loading CDNs";

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
  InfluxURL = influxUrl;
  toggleTop.checked = true;
  initMap(tileUrl);
  getCDNs();
  // getRegions();
}
