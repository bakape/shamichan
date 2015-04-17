Real-time imageboard.
MIT licensed.

**NOTE: The git master branch contains the development version of the
board. Crashes are not uncommon as new features are added and patched. For more stable revisions, please [download the latest release](https://github.com/bakape/meguca/releases).**

##Setup

* Install dependencies listed below
* Sign up for reCAPTCHA
* Run `npm install` to install npm deps and compile C++ addons
* Configure config.js, hot.js, imager/config.js and report/config.js
* Run `node builder.js` to run an auto-reloading development server

##Automatic cross-platform development setup:
* Install [VirtualBox](https://www.virtualbox.org/wiki/Downloads) and
[Vagrant](http://www.vagrantup.com/downloads.html)
* Open a shell in meguca's root directory and run `vagrant up`
* Grab a coffee
* Run `vagrant ssh` and `node builder`, once logged in. Your changes
will automatically sync both ways. [More info](https://www.vagrantup.com/)

##Production

* Have your webserver serve www/ (or wherever you've moved src, thumb, etc.)
* Run `node server/server.js` for just the server
* config.DAEMON support is broken for now
* You can update hot.js on-the-fly with `node server/kill.js`
* To remove compiled server dependancies run `make clean`
* Similarly `make client_clean` removes compiled client files

##Updating
* To recompile client code & CSS run `make client`. The new files can be
loaded into a running server with `node server/kill.js`
* After upgrading a node.js/io.js version or a meguca release run
`make upgrade` to recompile all dependancies

##Dependencies

* ImageMagick
* gcc, g++, make
* libpng with development headers
* [io.js](https://iojs.org) (Latest tested version is 1.7.1)
* redis

###Optional dependencies for various features

* ffmpeg 2.2+ for WebM support
  * with libmp3lame for MP3
* pngquant  2.3.0+ for PNG thumbnails

###Optional npm dependancies
* ~~daemon~~ (broken currently)
* node-expat for icecast now-playing banners
* send (if you want to serve static files directly from the node.js
process; useful in debug mode also)

## Documentation

* [docs/api.md](https://github.com/bakape/meguca/blob/master/docs/api.md) - JSON API documentation
* [docs/doushio.initscript.example](https://github.com/bakape/meguca/blob/master/docs/doushio.initscript.example) - example init script
* [docs/filemap.md](https://github.com/bakape/meguca/blob/master/docs/file_map.md.md) - project file map
* [docs/nginx.config.example](https://github.com/bakape/meguca/blob/master/docs/nginx.conf.example) - example ngingx reverse proxy configuration
file

##Standalone upkeep scripts

* upkeep/backup.js - uploads rdb to S3
* upkeep/backup.sh - MEGA backup script
* upkeep/clean.js - deletes archived images
* upkeep/purge_bans.sh - Removes all bans
* upkeep/purge_mod_sessions.sh - Closes all active moderator sessions
* upkeep/radio.js - icecast2 server integration
