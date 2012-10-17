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

* Have your webserver serve the static assets in `www`. Make sure `imager/config.js`'s `MEDIA_URL` corresponds to that URL.
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

* daemon
* icecast banners: node-expat, request
