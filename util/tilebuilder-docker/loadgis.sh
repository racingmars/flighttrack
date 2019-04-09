#!/bin/bash

set -xe

cd /tilebuild

if [ ! -f us-west-latest.osm.pbf ]
then
    wget https://download.geofabrik.de/north-america/us-west-latest.osm.pbf
fi

cd /opt/tilegen/openstreetmap-carto
osm2pgsql -C 24000 -s -G --hstore --style openstreetmap-carto.style --tag-transform-script openstreetmap-carto.lua -d gis /tilebuild/us-west-latest.osm.pbf

