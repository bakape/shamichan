Brief step by step guide to core meguca operations. Intended as a basic 
overview for porential devs.

## Building with `npm install`
- `node-gyp` compiles native C++ addons in `src/`
- npm downloads and installs project dependancies
- `scripts/bootstarp.js` checks for comfiguration files in `./config/` and 
copies over the examples in `config/examples`, if none
- gulp builds client files (see `gulpfile.js`)
	- CSS themes: `less/` -> `www/css/`
	- Language packs: `lang/` -> `www/js/lang/`
	- Moderation client module: `client/mod/` -> `state/mod.js`
	- Dependancies: `.node_modules/` & `lib/` -> `www/js/vendor.js`
	- Standard client for up-to-date browsers: `client/` -> `www/js/client.js`
	- Legacy client for old/shitty/hipster browsers: `client/` -> 
	`www/js/legacy/client.js`

## Server starting with `npm start`
- npm calls `node index.js`
- `server/index` parses arguments, sets up logging and ES6 transpilation 
through Babel.js and requires the actual server - `server/server`
- `server/state` reads `config/hot` (workaround for module dependancy hierarchy)
- `db` initialises global redis connection
- `imager` is preloaded and redied to for operation
- `server/state`
	- loads `config/hot` again
	- loads `state/mod.js` moderation client module into memory
	- hashes client resources for transparent versioning and transition
	- builds HTML templates from `tmpl/` for each language
- `db` reads redis for thread and board numbers and populates caches
- `server/web` starts listening for requests

## Page rendering
- client sends a requests
- `server/web/index` validates client and authenticates staff
- `server/web/html`
	- checks client elegebility to access page
	- instantiates `db.Yakusoku()` (which instantiates `db.Reader()`)
		- reads thread data and emits it as the `begin` event
		- reads each thread and post and emits relevant events for each
	- instantiates the appropriate class from `server/render`
		- instantiates  a `common/oneesama()` rendering singleton
		- parses cookies and configures `oneeSama()` for that particular client
		- handles `Reader()`'s events and calls apropriate `OneeSama()` methods
		- eventfully concatenates `OneeSama()` output with prerendered 
		templates from `server/state` and streams them part by part to the 
		client
		
## Staring the client
- `www/js/setup.js` reads options from localStorage and sets the CSS theme
- `www/js/loader.js`
	- loads the appropriate language bundle
	- determines, if browser has ES6 support, and loads either `www/js/client.js`
	or `www/js/legacy/client.js`
	- if authenticated staff, loads moderation bundle from `../mod.js`
	- starts the client (see `client/main`)
