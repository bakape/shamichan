FROM postgres:11

RUN mkdir -p /docker-entrypoint-initdb.d
COPY init.sh /docker-entrypoint-initdb.d/
COPY init.sql /docker-entrypoint-initdb.d/
