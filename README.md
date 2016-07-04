[![GoDoc](https://godoc.org/github.com/bakape/meguca?status.svg)](https://godoc.org/github.com/bakape/meguca)
[![Build Status](https://travis-ci.org/bakape/meguca.svg)](https://travis-ci.org/bakape/meguca)
[![Dependency Status](https://david-dm.org/bakape/meguca.svg)](https://david-dm.org/bakape/meguca)

__Note: The version in the master branch is currently in early development.
For deploying a production-ready legacy meguca instance, download the [latests
release](https://github.com/bakape/meguca/releases/latest) and consult the
bundled README.__

Platforms: Linux, OSX, Win64

License: GNU AGPL

##Runtime dependencies
* [RethinkDB](https://rethinkdb.com/docs/install/).
On Linux RethinkDB does not enable a configuration file by default. If you don't
want to configure anything, just copy `/etc/rethinkdb/default.conf.sample` into
`/etc/rethinkdb/instances.d/instance1.conf`.
* ffmpeg >= 3.0 libraries (libavcodec, libavutil, libavformat, libswscale)
compiled with:
	* libvpx
	* libvorbis
	* libopus
	* libtheora
	* libx264
	* libmp3lame

##Compiled Binaries
Compiled binary release archives for linux/x86_64 and windows/x86_64 are downloadable
from the [release](https://github.com/bakape/meguca/releases) page.

##Production
* See `./meguca help` for server operation
* Login into the "admin" with the password "password" and change the password
* For upgrading between semver major releases see `docs/migration.md`

##Building from source

###All Platforms
* Install:
	* [Go](https://golang.org/doc/install) >=1.6
	* [Node.js](https://nodejs.org) >=5.0 (for building the client)

###Linux and OSX
* Install:
    * GCC or Clang
    * make
    * ffmpeg >= 3.0 development libraries (libavcodec, libavutil,
    libavformat, libswscale)
    * git
    * zip
* Run `make`

###Windows
* Install [MSYS2](https://sourceforge.net/projects/msys2/)
* Open MSYS2 shell
* Install with pacman:
    * mingw-w64-x86_64-ffmpeg
    * mingw-w64-x86_64-gcc
    * mingw-w64-x86_64-pkg-config
    * git
    * make
    * zip
* Navigate to the meguca root directory
* Run `make`

##Development
* `./meguca` or `./meguca debug` to run the server in development mode
* `make server` and `make client` build the server and client separately
* `make watch` watches the file system for changes and incrementally rebuilds
the client
* `make clean` removes files from the previous compilation
* `make dist_clean` in addition to the above, removes configuration, images and
stored assets

###Linux only
* make creates a Go workspace in the `.build` subdirectory. If you don't have a
proper Go worksapce set up, you can simply `export GOPATH=$(pwd)/.build` to
temporarily assign `.build` as your Go workspace in the current shell. This will
allow you to use native go commands such as `go test` and `go build`.

// TODO: Rewrite everything below for v2

## Documentation
* docs/api.md - JSON API spec
* docs/dev_guide.md - Brief description on project operation for developers
* docs/migration_*.js - Major semvser migration instructions
* docs/doushio.initscript.example - Init script example
* docs/nginx.config.example - ngingx configuration example

##Standalone upkeep scripts
* scripts/backup.js - uploads rdb to S3
* scripts/backup.sh - MEGA backup script
* scripts/purge_bans.sh - Removes all bans
* scripts/purge_mod_sessions.sh - Closes all active moderator sessions
* scripts/radio.js - icecast2 server integration
* scripts/send.js - global websocket push messages
