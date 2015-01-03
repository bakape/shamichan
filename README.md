Real-time imageboard.
MIT licensed.

Note: The git master branch contains the development version of the board. For more stable revisions, please download the latest release.

Setup:

* Install dependencies listed below
* Sign up for reCAPTCHA
* Copy config.js.example to config.js and configure
* Copy hot.js.example to hot.js and configure
* Copy imager/config.js.example to imager/config.js and configure
* Copy report/config.js.example to report/config.js and configure
* Run `npm install` to install npm deps and compile a few helpers
* Run `node builder.js` to run an auto-reloading development server
* Alternatevly, you can run `curl -Ls https://raw.githubusercontent.com/bakape/doushio/master/DEPLOY.sh | bash -`
	as root for a basic semiautomatic production setup. Currently Ubuntu-only

Production:

* Have your webserver serve www/ (or wherever you've moved src, thumb, etc.)
* Run `node server/server.js` for just the server
* config.DAEMON support is broken for now
* You can update client code & hot.js on-the-fly with `node server/kill.js`
* For nginx hosting/reverse proxying, refer to docs/nginx.conf.example
* For a sample init script, refer to docs/doushio.initscript.example

Dependencies:

* ImageMagick
* libpng
* node.js + npm
* redis
* ffmpeg 2.2+ if supporting WebM
* pngquant  2.3.1+ for PNG thumbnails
* exiftool, if you want to strip images of their exif (must be enabled in config.js)

Optional npm deps for various features:

* ~~daemon~~ (broken currently)
* icecast now-playing banners: node-expat
* [send](https://github.com/visionmedia/send) (if you want to serve static files directly from the node.js process; useful in debug mode also)
* forced anonymity hours: cron

Standalone upkeep scripts:

* archive/daemon.js - moves old threads to the archive
* upkeep/backup.js - uploads rdb to S3
* upkeep/clean.js - deletes archived images
* upkeep/radio.js - icecast2 server integration

Project map:

* __admin/__
	* __client.js__		Client-side moderation interface and logic
	* __common.js__		Functions common to all admin modules
	* __index.js__		Server-side communication and database handler
	* __panel.js__		Renders the administrator panel
* __anonhours/__
	* __server.js__		Anonymity and random name hour logic
* __archive/__
	* __daemon.js__		Archiver/thread deletion daemon
	* __dump.js__		Static archiver. Currently not ready for use
* __autojoe/__
	* __server.js__		Automatic poster name assignemt by parsing r/a/dio song title
* __client/__
	* __amusement.js__	Hooks into the server for hash commands etc.
	* __client.js__		Main client module. Handles parsing server communication
	* __conn.js__		Maintains and prints websocket connection status
	* __drop.js__		Image drag-and-drop upload
	* __embed.js__		Embeds Youtube and Soundcloud links
	* __extract.js__	Extracts models from server-rendered threads and posts
	* __fun.js__		Is loaded in `fun_thread`, set by an Admin
	* __gravitas.js__	1000000th post congratulatory formatting
	* __hide.js__		Hide user-set posts and threads
	* __hover.js__		Hover post link preview and Youtube/Soundcload embed expansion
	* __init.js__		Initialise client
	* __memory.js__		LocalStorage memory controller
	* __menu.js__		Post actions menu (the in the upper left corner)
	* __models.js__		Backbone models
	* __notify.js__		(YOU), unread count in tab title and desktop notifications
	* __options.js__	User options and options-panel rendering
	* __posting.js__	Post form logic
	* __scroll.js__		Page scrolling and lock-to-bottom
* __curfew/__
	* __client.js__		Performs the client-side DOM teardown
	* __server.js__		Closes access to boards on time-based configuration
* __docs/__
	* __doushio.initscript.example__	Sample init script
	* __nginx.conf.example__			Sample nginx configuration file
* __imager/__
	* __config.js.example__	Sample image & video processing configuration
	* __daemon.js__			Recieves and processes images and video from clients
	* __db.js__				Handles Redis image keys
	* __finapng.c__			Compiles into APNG detection utility on build
	* __ndex.js__			Handles various image-related structured information
	* __jobs.js__			Image & Video processing scheduler
	* __Makefile__			Compiles findapng.c and perceptual.c
	* __perceptual.c__		Compiles into perceptual hash generator on build
* __report/__
	* __client.js__			Renders report panel
	* __config.js.example__	Sample reports configuration
	* __server.js__			Dispatches reports as emails
* __server/__
	* __amusement.js__	Hash commands and other server-side hooks
	* __caps.js__		Handles board access and moderation/administration rights
	* __kill.js__		Reloads hot-reloadable resources
	* __msgcheck.js__	Validate objects recieved through websocket
	* __okyaku.js__		Handles websocket connections
	* __opt.js__		Handles various configuration options
	* __persona.js__	Authenticates moderators/admins
	* __render.js__		Handles initial server-side render. All future rendering is done client-side
	* __server.js__		Primary server module. Handles communication and passes requests to other modules
	* __state.js__		Loads/reloads hot-reloadable configuration and expands templates
	* __web.js__		HTTP web server
* __state/__
	* __scripts.json__	Contains debugging samples of JS scipts, that are served to the client
* __time/__
	* __client.js__		Corrects post timestamps for timezones
	* __server.js__		Server-side hooks for time correction
* __tmpl/__
	* __alookup.html__		Foolz archive redirect template
	* __curfew.html__		Is served, when board is closed due to curfew
	* __filter.html__		No clue
	* __index.html__		Primary template for boards and threads
	* __login.html__		Template for the /login/ Persona login path
	* __redirect.html__		Hash URL redirecion
	* __suspension.html__	Ban notice template. Currently disfunctional and redundant
* __tripcode/__
	* __Makefile__		A makefile
	* __binding.gyp__	Node bindings for the tripcode hasher
	* __tripcode.cc__	Compiles into tripcode hash utility on build
	* __wscript__		Builds node tripcode module
* __upkeep/__
	* __backup.js__		Amazon S3 backup script
	* __backup.sh__		MEGA backup script
	* __clean.js__		Deletes archived thread images
	* __poll.xsl__		Icecast configuration file
	* __purge_bans.sh__	Removes all bans
	* __radio.js__		Icecast polling and now-playing-banner updater
* __www/__				This folder is to be served by a webserver on default configuration
	* __css/__
		* __ui/__					Global user interface assets
		* __base.css__				Global board and thread CSS
		* __gravitas.css__			100000th post CSS
		* __mod.css__				Moderation interface CSS
		* __persona-buttons.css__	CSS for logging in with 'misaki' in the email field
		* __the rest __				various theme CSS and assets
	* __403.html__				Custom 403 page
	* __404.html__				Custom 404 page
	* __50x.html__				Custom blame-the-devs page
	* __MadokaRunes.ttf__		Font for 404 and 403 pages
	* __favicon.ico__			A favicon
	* __kana__				Symlink to default spoiler directory (../assets/kana)
	* __maintenance.jpg__	403 and 404 background image
* __DEPLOY.sh__				Semi-automatic server deployment script
* __builder.js__			Auto-reloading development server
* __common.js__				Contains generic logic for building HTML both server- and client-side
* __config.js.example__		Sample global configuration file
* __db.js__				Handles Redis writes and listeners
* __deps.js__			Lists dependancies
* __etc.js__			Various helper functions
* __hooks.js__			Hooks for server-client and viseversa code execution
* __hot.js.example__	Sample hot-reloadable configurations file
* __make_client.js__	Generates the main client javascript files to be served
* __package.json__		NPM configuration
* __pipeline.js__		Creates and hashes client-side javascript files
* __tail.js__			No clue
