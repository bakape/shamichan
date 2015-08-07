Real-time imageboard.
MIT licensed.
Supported platforms: Linux, OS X (win64 pending)

**NOTE: The git master branch contains the development version of the board.
Crashes are not uncommon as new features are added and patched. The database 
structure and transport API are in active development. Don't use the master 
branch in production. For more stable revisions, please 
[download the latest release](https://github.com/bakape/meguca/releases).**

##Setup
* Install dependencies listed below
* Sign up for reCAPTCHA
* Run `npm install` to install npm deps and compile C++ addons
* Configure the files in `config/`
* Run `node builder.js` to start an auto-reloading development server

##Production
* Have your webserver serve `www/`
  * It is highly recommended to use a dedicated webserver for serving static
  files and as a reverse proxy. Even if you choose to use the default inbuilt
  webserver, you still need to set `MEDIA_URL` in `config/imager` for image
  search links to work.
* Run `npm start` to start the server
* You can update `config/hot.js` on-the-fly with `node server/kill.js`
* To remove compiled server dependancies run `make clean`
* Similarly `make client_clean` removes compiled client files

##Updating
* To recompile client JS & CSS run `make client`. The new files can be loaded
 into a running server with `node server/kill.js`
* After upgrading an io.js version or a meguca release run `make upgrade` to 
recompile all dependancies

##Dependencies
* [node-gyp dependancies](https://github.com/TooTallNate/node-gyp/#installation)
* imagemagick
* libpng with development headers
* [io.js](https://iojs.org) >=2.0.0
* redis

###Optional dependencies
* ffmpeg 2.2+ with libvpx, libvorbis and libopus for WebM support
  * with libmp3lame for MP3
* pngquant  2.3.0+ for PNG thumbnails

## Documentation
* [JSON API: docs/api.md](https://github.com/bakape/meguca/blob/master/docs/api.md)
* [init script example: docs/doushio.initscript.example](https://github.com/bakape/meguca/blob/master/docs/doushio.initscript.example)
* [ngingx configuration example: docs/nginx.config.example](https://github.com/bakape/meguca/blob/master/docs/nginx.conf.example)

##Standalone upkeep scripts
* scripts/backup.js - uploads rdb to S3
* scripts/backup.sh - MEGA backup script
* scripts/purge_bans.sh - Removes all bans
* scripts/purge_mod_sessions.sh - Closes all active moderator sessions
* scripts/radio.js - icecast2 server integration
* scripts/send.js - global websocket push messages
