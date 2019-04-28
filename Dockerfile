FROM debian:stable

EXPOSE 8000

# Install OS dependencies
RUN apt-get update
RUN apt-get install -y apt-utils build-essential pkg-config libpth-dev libavcodec-dev libavutil-dev libavformat-dev libswscale-dev libwebp-dev libopencv-dev libgeoip-dev git lsb-release wget curl sudo
RUN apt-get dist-upgrade -y

# Make the "en_US.UTF-8" locale so postgres will be utf-8 enabled by default
RUN apt-get install -y locales
RUN rm -rf /var/lib/apt/lists/*
RUN localedef -i en_US -c -f UTF-8 -A /usr/share/locale/locale.alias en_US.UTF-8
ENV LANG en_US.utf8

# Install PostgreSQL
RUN echo deb http://apt.postgresql.org/pub/repos/apt/ $(lsb_release -sc)-pgdg main >> /etc/apt/sources.list.d/pgdg.list
RUN wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
RUN apt-get update
RUN apt-get install -y postgresql

# Increase PostgreSQL connection limit by changing `max_connections` to 1024
RUN sed -i "/max_connections =/d" /etc/postgresql/11/main/postgresql.conf
RUN echo max_connections = 1024 >> /etc/postgresql/11/main/postgresql.conf

# Create role and DB
USER postgres
# Needs to be done in one step, so the service is guaranteed to run for DB and
# role creation
RUN service postgresql start && \
	until pg_isready > /dev/null; do sleep 1; done && \
	psql -c "CREATE USER meguca WITH LOGIN PASSWORD 'meguca' CREATEDB" && \
	createdb -T template0 -E UTF8 -O meguca meguca
USER root

# Install Go
RUN wget -q -O- https://dl.google.com/go/go1.12.4.linux-amd64.tar.gz | tar xpz -C /usr/local
ENV PATH="${PATH}:/usr/local/go/bin"

# Install Node.js
RUN wget -q -O- https://deb.nodesource.com/setup_10.x | bash -
RUN apt-get install -y nodejs

# Copy and build meguca
RUN mkdir -p /meguca
WORKDIR /meguca
ENTRYPOINT ["/meguca/scripts/with_postgres.sh"]
CMD ["./meguca", "-a", ":8000"]
COPY . .
RUN make
