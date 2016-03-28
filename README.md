##Setup
* Install [dependencies](#dependencies) listed below
* Sign up for reCAPTCHA
* Run `npm install` to install npm deps and compile C++ addons
* Configure the files in `config/`
* Run `node builder.js` to start an auto-reloading development server

##Cross-platform development with Vagrant
* Install [VirtualBox](https://www.virtualbox.org/wiki/Downloads) and
[Vagrant](http://www.vagrantup.com/downloads.html)
* Open a shell in meguca's root directory and run `vagrant up`
* Grab a coffee
* Run `vagrant ssh` and `node builder`, once logged in. Your changes will
automatically sync both ways. [More info](https://www.vagrantup.com/)

##Production
* Have your webserver serve `www/`
  * It is highly recommended to use a dedicated webserver for serving static
  files and as a reverse proxy. Even if you choose to use the default inbuilt
  webserver, you still need to set `MEDIA_URL` in `config/imager` for image
  search links to work.
* Run `npm start/stop/restart` to start/stop/restart the server
* You can update `config/hot.js` and client files without restarting the server
with `node server/kill.js`
* All errors are logged to `./error.log`

##Updating
* To recompile the project rerun `npm install`
* After upgrading an node.js version also run `npm rebuild`
* See `docs/` for upgrading between semver major releases

##Dependencies
* node.js >=5.0.0
* [node-gyp dependancies](https://github.com/TooTallNate/node-gyp/#installation)
* imagemagick
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
