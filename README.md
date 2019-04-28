[![GoDoc](https://godoc.org/github.com/bakape/meguca?status.svg)](https://godoc.org/github.com/bakape/meguca)
[![Build Status](https://travis-ci.org/bakape/meguca.svg?branch=master)](https://travis-ci.org/bakape/meguca)

# meguca
anonymous realtime imageboard focused on high performance, free speech and transparent moderation

Platforms: Linux, OSX, Win64

License: GNU AGPL

## Runtime dependencies
* [PostgresSQL](https://www.postgresql.org/download/) >= 10.0

## Building from source


### Native installation.

For installing meguca directly onto a server follow the steps bellow.
A reference list of commands can be found in `./docs/installation.md`

#### Build dependencies
* [Go](https://golang.org/doc/install) >=1.11 (for building server)
* [Node.js](https://nodejs.org) >=5.0 (for building client)
* C11 compiler
* make
* pkg-config
* pthread
* ffmpeg >= 3.1 libraries (libavcodec, libavutil, libavformat, libswscale)
compiled with:
    * libvpx
    * libvorbis
    * libopus
    * libtheora
    * libx264
    * libmp3lame
* OpenCV >= 2
* libgeoip
* git

#### Linux and OSX
* Run `make`

#### Windows
* Install [MSYS2](https://sourceforge.net/projects/msys2/)
* Open MSYS2 shell
* Install dependencies listed above with the `mingw-w64-x86_64-` prefix with
pacman
* Navigate to the meguca root directory
* Run `make`

### Docker

Meguca can be deployed in a self-contained Docker container. Install [Docker](https://www.docker.com/)
and run
```
docker build -t meguca .
docker run -t -p 8000:8000 meguca
```

## Setup
* See `./meguca help` for server operation
* Login into the "admin" account via the infinity symbol in the top banner with
the password "password"
* Change the default password
* Create a board from the administration panel
* Configure server from the administration panel
* To enable country flags on posts download and place `GeoLite2-Country.mmdb`
into the root directory
* To avoid having to always type in CLI flags on server start you can specify them in `config.json` file in the project root. A sample file with all the default settings can be found in `docs/`.

## Development

* See `./docs` for more documentation
* `./meguca` or `./meguca debug` run the server in development mode
* `make server` and `make client` build the server and client separately
* `make watch` watches the file system for changes and incrementally rebuilds
the client
* `make clean` removes files from the previous compilation
* `make {test,test_no_race,test_docker}` run regular, without data race
detection and Dockerized test suites, respectively
* To run server unit tests (unless Dockerized) add database creation rights to
your PostgreSQL role
