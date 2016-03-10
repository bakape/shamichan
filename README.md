[![GoDoc](https://godoc.org/github.com/bakape/meguca/server?status.svg)](https://godoc.org/github.com/bakape/meguca/server) [![Build Status](https://travis-ci.org/bakape/meguca.svg)](https://travis-ci.org/bakape/meguca) [![Dependency Status](https://david-dm.org/bakape/meguca.svg)](https://david-dm.org/bakape/meguca)

__Note: The version in the master branch is currently in early development.
For deploying a production-ready legacy meguca instance, download the [latests
release](https://github.com/bakape/meguca/releases/latest) and consult the
bundled README.__

Platforms: Linux, OSX, Win64(soon™)

##Runtime dependencies
* [RethinkDB](https://rethinkdb.com/docs/install/)
    * RethinkDB does not enable a configuration file by default. If you don't
    want to configure anything, just copy `/etc/rethinkdb/default.conf.sample`
	into `/etc/rethinkdb/instances.d/instance1.conf`. You might also set it to
	autostart on boot. See the
    [official guide](http://www.rethinkdb.com/docs/start-on-startup/).

##Installable binaries
Coming soon™

##Building from source
* Install:
    * GCC or Clang
    * make
    * [Go](https://golang.org/doc/install)
    * [Node.js](https://nodejs.org) (required for building the client)
* Run `make`
* Prepare server for operation by running `make init`

##Production
* Edit `./config/config.json` to configure your instance
* See `./meguca help` for server daemon control
* For upgarding between semver major releases see `docs/migration.md`

##Development
* `./meguca debug` to run the server in development mode
* `make server` and `make client` build the server and client separately
* `make watch` watches the file system for changes and incrementally rebuilds
the client
* `make clean` removes files from the previous compilation
* `make dist_clean` in addition to the above, removes configuration, images and
stored assets

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
