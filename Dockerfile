FROM golang:bullseye

EXPOSE 8000

RUN mkdir -p /meguca/images
ENTRYPOINT ["./scripts/with_postgres.sh"]
CMD ["./meguca", "-a", ":8000"]
WORKDIR /meguca

# Install OS dependencies
ENV DEBIAN_FRONTEND=noninteractive
RUN echo deb-src \
	http://ftp.debian.org/debian/ \
	bullseye main contrib non-free \
	>> /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y \
	build-essential \
	pkg-config \
	libpth-dev \
	libwebp-dev \
	libopencv-dev \
	libgeoip-dev geoip-database \
	python3 python3-requests \
	git lsb-release wget curl netcat postgresql-client gzip
RUN apt-get dist-upgrade -y

# Build a known working version of FFmpeg without thumnailer crashing.
#
# Will investigate cause and fix either thumbnailer or FFmpeg code at a later
# date.
#
# Using RUN directives caches more readily than a script.
RUN apt-get build-dep -y ffmpeg
RUN mkdir /src
RUN git clone \
	--branch release/4.1 \
	--depth 1 \
	https://github.com/FFmpeg/FFmpeg.git \
	/src/FFmpeg
WORKDIR /src/FFmpeg
RUN ./configure
RUN nice -n 19 make -j $(nproc)
RUN make install
WORKDIR /meguca

# Install Node.js
RUN wget -q -O- https://deb.nodesource.com/setup_16.x | bash -
RUN apt-get install -y nodejs

# Cache dependency downloads, if possible
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY package.json .
COPY package-lock.json .
RUN npm install --include=dev

# Copy and build meguca
COPY . .

COPY docs/config.json .
RUN sed -i 's/localhost:5432/postgres:5432/' config.json
RUN sed -i 's/"reverse_proxied": false/"reverse_proxied": true/' config.json
RUN sed -i 's/127\.0\.0\.1:8000/:8000/' config.json

RUN make server
RUN make client
