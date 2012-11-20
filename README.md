Real-time imageboard.
MIT licensed.

Setup:

* Install deps and npm deps
* Copy config.js.example to config.js and configure
* Copy hot.js.example to hot.js and configure
* Copy imager/config.js.example to imager/config.js and configure
* Run `make` to build some essential components
* Run `node builder.js` to run an auto-reloading development server

Production:

* Run `node server/server.js` for just the server
* `config.DAEMON` support is broken for now.
* Be sure to `make client` for any client-affecting change

Dependencies:

* ImageMagick
* node.js
* redis

npm modules:

* async
* formidable
* redis
* sockjs
* winston

Optional npm deps for various features:

* ~~daemon~~ (broken currently)
* icecast now-playing banners: node-expat, request
* [send](https://github.com/visionmedia/send) (if you want to serve static files directly from the node.js process; useful in debug mode also)
