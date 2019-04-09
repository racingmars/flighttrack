Tile Rendering
==============

The files in this directory allow you to create a Docker image that you may
then run to render map tiles for the web application. Using locally-served
tiles makes the map more responsive than using OpenStreetMap's servers.

Step 1 - Customize for your needs
---------------------------------

The files setup.sh, postgres_settings.conf, and loadgis.sh will be baked into
the Docker image, so you should make any customizations to those files before
building the image.

setup.sh: This file should not require customizations.

postgres_settings.conf: The contents of this file was generated with
  [https://pgtune.leopard.in.ua/]. The values in there are based on Postgres
  using 64GB of RAM and 8 CPUs. Adjust for your computer accordingly.

loadgis.sh: This is the script that you will use inside the container to
  download and load the OpenStreetMap data into the database running inside
  the container. I have it set to download the western United States.

  Also note the -C 24000 argument to osm2pgsql. This sets the cache size to
  24,000MB of RAM. Adjust appropriately.

  It's not absolutely critical that you change this file right now -- it
  will be loaded into the image, but you'll be able to edit it inside the
  image if you wish.

Step 2 - Create Docker image
----------------------------

Build the Docker image, tilebuilder:latest, by running the build.sh script.
This will take a while.

Step 3 - Run the Docker image
-----------------------------

After the image is created, run it. Use a volume mapping to map a directory
on the host to /tilebuild in the container. This is where the OSM data will
be downloaded (so it can be re-used) and where the tiles will be created.

Run the image with a command like:
  docker run -d -v /home/me/tilebuild:/tilebuild tilebuilder:latest

The container is now running.

Step 4 - Load the cartography data into the database
----------------------------------------------------

Exec a shell in the running container. This will connect as the postgres user,
which keeps things easy inside the container for connecting to Postgres.

  docker exec -it <container name or id> /bin/bash

You should be in the /opt/tilegen directory. Run ./loadgis.sh and wait.
(If you need to change the region, but didn't do so before building the
Docker image, you should edit the file in the container before running
it. Vim is installed.)

Step 5 - Generate the tiles
---------------------------

Edit the generate_tiles.py file in the container. The bottom of the file has
a bunch of example render function calls. Delete them all or comment them all
out, and make one for your desired bounding box and zoom level.

For example, I used:
  minZoom = 7
  maxZoom = 15
  bbox = (-125.67, 42.08, -117.1, 48.42)
  render_tiles(bbox, mapfile, tile_dir, minZoom, maxZoom, "PNW")

When you're ready to run the file, `cd openstreetmap-carto`, then run with:
MAPNIK_MAP_FILE=osm.xml  MAPNIK_TILE_DIR=/tilebuild/tiles/ python ../generate_tiles.py

Wait a while. Mine took about eight hours:
  real  479m14.258s
  user  1401m20.988s
  sys   36m23.040s

Step 6 - Done!
--------------

You should now have your tiles in /tilebuild/tiles, also accessible on the
host wherever the volume mapping is. Get those tiles into web/static/tiles
and you're ready to go!

References / Credits
--------------------

[https://wiki.openstreetmap.org/wiki/Creating_your_own_tiles]

My setup.sh script started life based on Mark Meyer's project:
[https://ofosos.org/2018/11/04/osm-tile-creation-on-aws-spot/]

Font installation updated based on:
[https://ircama.github.io/osm-carto-tutorials/kosmtik-ubuntu-setup/#install-the-fonts-needed-by-openstreetmap-carto]
