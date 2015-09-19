Real-time imageboard.
MIT licensed.
Supported platforms: Linux, OS X (win64 pending)

**NOTE: The git master branch contains the development version of the board.
Crashes are not uncommon as new features are added and patched. Don't use the
master branch in production. For more stable revisions, please [download the
 latest release](https://github.com/bakape/meguca/releases).**

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
* You can update `config/hot.js` and client files without restarting the server with `node server/kill.js`
* All errors are logged to `./error.log`

##Updating
* To recompile the project rerun `npm install`
* After upgrading an io.js version also run `npm rebuild`
* See `docs/` for upgrading between semver major releases

##Dependencies
* [node-gyp dependancies](https://github.com/TooTallNate/node-gyp/#installation)
* imagemagick
* node.js >=2.0.0
* redis

###Optional dependencies
* ffmpeg 2.2+ with libvpx, libvorbis and libopus for WebM support
  * with libmp3lame for MP3
* pngquant  2.3.0+ for PNG thumbnails

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
