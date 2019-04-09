#!/bin/bash

set -xe

export DEBIAN_FRONTEND=noninteractive

# Initial prereqs
apt-get update
apt-get install -y curl wget gnupg software-properties-common sudo

# Set up PostgreSQL repo
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
echo "deb http://apt.postgresql.org/pub/repos/apt/ bionic-pgdg main" > /etc/apt/sources.list.d/pgdb.list

# Set up NodeJS repo
curl -sL https://deb.nodesource.com/setup_10.x | sudo -E bash -

apt-get update

# Install everything
apt-get -y install \
	postgresql-11 postgresql-11-postgis-2.5 \
	osm2pgsql \
	unzip python git \
	nodejs  \
	libmapnik-dev python-mapnik mapnik-utils \
	ttf-unifont fonts-noto-cjk fonts-noto-hinted fonts-noto-unhinted \
	fonts-hanazono fontconfig \
    vim

/etc/init.d/postgresql start

sudo -u postgres createuser gisuser
sudo -u postgres createdb --encoding=UTF8 --owner=gisuser gis
sudo -u postgres psql --username=postgres --dbname=gis \
	-c "CREATE EXTENSION postgis;"
sudo -u postgres psql --username=postgres --dbname=gis \
	-c "CREATE EXTENSION postgis_topology;"
sudo -u postgres psql --username=postgres --dbname=gis \
	-c "CREATE EXTENSION hstore;"

mkdir -p /opt/tilegen
cd /opt/tilegen

wget https://svn.openstreetmap.org/applications/rendering/mapnik/generate_image.py
wget https://svn.openstreetmap.org/applications/rendering/mapnik/generate_tiles.py

# INSTALL carto
npm -g install carto

git clone https://github.com/gravitystorm/openstreetmap-carto

cd openstreetmap-carto

carto project.mml >osm.xml

# Avoid mapnik error message about this font missing; it's now named "Unifont
# Medium" with uppercase U, which is also listed in the osm.xml file.
sed -i 's^<Font face-name="unifont Medium" />^^' osm.xml

./scripts/get-shapefiles.py

# INSTALL fonts
cd /tmp
mkdir noto
cd noto
git clone --depth 1 https://github.com/googlei18n/noto-emoji.git
git clone --depth 1 https://github.com/googlei18n/noto-fonts.git
cp noto-emoji/fonts/NotoColorEmoji.ttf /usr/share/fonts/truetype/noto
cp noto-emoji/fonts/NotoEmoji-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansArabicUI-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoNaskhArabicUI-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansArabicUI-Bold.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoNaskhArabicUI-Bold.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansAdlam-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansAdlamUnjoined-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansChakma-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansOsage-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansSinhalaUI-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansArabicUI-Regular.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansCherokee-Bold.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansSinhalaUI-Bold.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansSymbols-Bold.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/hinted/NotoSansArabicUI-Bold.ttf /usr/share/fonts/truetype/noto
cp noto-fonts/unhinted/NotoSansSymbols2-Regular.ttf /usr/share/fonts/truetype/noto

fc-cache -fv

/etc/init.d/postgresql stop

rm -rf /tmp/*
chown -R postgres:postgres /opt/tilegen
