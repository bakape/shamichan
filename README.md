Real-time imageboard.
MIT licensed.

Setup:

* Install dependencies listed below
* Copy config.js.example to config.js and configure
* Copy hot.js.example to hot.js and configure
* Copy imager/config.js.example to imager/config.js and configure
* Run `npm install` to install npm deps and compile a few helpers
* Run `node builder.js` to run an auto-reloading development server

Production:

* Have your webserver serve www/ (or wherever you've moved src, thumb, etc.)
* Run `node server/server.js` for just the server
* config.DAEMON support is broken for now
* Be sure to `make client` for any client-affecting change

Dependencies:

* ImageMagick
* node.js
* redis

Optional npm deps for various features:

* ~~daemon~~ (broken currently)
* icecast now-playing banners: node-expat
* [send](https://github.com/visionmedia/send) (if you want to serve static files directly from the node.js process; useful in debug mode also)
