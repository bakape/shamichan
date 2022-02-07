[![GoDoc](https://godoc.org/github.com/bakape/shamichan?status.svg)](https://godoc.org/github.com/bakape/shamichan)
[![Build](https://github.com/bakape/shamichan/workflows/Build/badge.svg?branch=master)](https://github.com/bakape/shamichan/actions?query=workflow%3ABuild+branch%3Amaster)

# shamichan
anonymous realtime imageboard focused on high performance, free speech and transparent moderation

Platforms: Linux, Docker

License: GNU GPLv3


__The master branch is currently undergoing active breaking changes towards shamichan v7. If you intend to deploy shamichan, please use the v6 branch.__

## Runtime dependencies

* [PostgresSQL](https://www.postgresql.org/download/) >= 10.0

## Docker

Shamichan can be deployed in a self-contained [Docker](https://www.docker.com/)
container.

Simply run

```
docker run --restart unless-stopped -d --network host --mount type=bind,source=$image_dir,destination=/shamichan/images bakape/shamichan -d $DB
```

where:
- $image_dir is the directory to store uploaded images
- $DB is the URL to a PostgreSQL database to use for data storage

## Building from source

### Native installation.

For installing shamichan directly onto a server follow the steps bellow.
A reference list of commands can be found in `./docs/installation.md`

#### Build dependencies

* [Go](https://golang.org/doc/install) >=1.13 (for building server)
* [Node.js](https://nodejs.org) >=12.0 (for building client)
* [Rust](https://www.rust-lang.org/) >= 1.41
* C11 compiler
* make
* pkg-config
* pthread
* ffmpeg >= 4.1 libraries (libavcodec, libavutil, libavformat, libswscale)
compiled with:
    * libvpx
    * libvorbis
    * libopus
    * libtheora
    * libx264
    * libmp3lame
* OpenCV 2-3
* libgeoip
* OpenSSL
* git

NB: Ubuntu patches to ffmpeg on some Ubuntu versions <19.10 break image
processing. In that case please compile from unmodified ffmpeg sources using:

```
sudo apt build-dep ffmpeg
git clone https://git.ffmpeg.org/ffmpeg.git ffmpeg
cd ffmpeg
git checkout n4.1
./configure
make -j`nproc`
sudo make install
```

#### Compilation

Run `make`

## Setup

### Deployment

<!-- TODO: update -->

shamichan can be started in debug mode simply with `./shamichan`.
Configurations are split between shamichan instance configurations
and server instance configurations, which are required to start
the server and connect to the database.
The shamichan instance configurations are stored in the database, but
server instance configurations are optionally loaded from a `config.json`
file on server start.
A sample configuration file can be found under `docs/config.json`.
Documentation for this file is available under `docs/config.jsonc`.

It is recommended to serve shamichan behind a reverse proxy like NGINX or Apache
with properly configured TLS settings. A sample NGINX configuration file can be
found in `docs/`.

### Initial instance configuration

* TODO: grant first user to go to a URL admin rights

## Development

TODO: note about compile-time query checking and how to support it

* See `./docs` for more documentation
* `make install_tools` to install required build tools
* `make server` and `make client` build the server and client separately
* Pass `DEBUG=1` before make command to build in debug mode
* Pass `NO_DEPS=1` before make command to not install dependencies with npm on
each build
* `make clean` removes files from the previous compilation
* `make {test,test_no_race}` run regular test and tests without data race
detection, respectively
* To run server unit tests add database creation rights to your PostgreSQL role
