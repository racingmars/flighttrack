FROM ubuntu:bionic
WORKDIR /setup
COPY setup.sh /setup
COPY loadgis.sh /opt/tilegen
COPY postgres_settings.conf /etc/postgresql/11/main/conf.d
RUN /setup/setup.sh
WORKDIR /opt/tilegen
VOLUME ["/tilebuild"]
USER postgres
CMD ["/usr/lib/postgresql/11/bin/postgres", "-D", "/var/lib/postgresql/11/main", "-c", "config_file=/etc/postgresql/11/main/postgresql.conf"]
