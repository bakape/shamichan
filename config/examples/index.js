module.exports = {
	LISTEN_PORT: 8000,
// Host address to listen on. Use null for localhost
	LISTEN_HOST: null,
// Debuging mode. Do not use in production
	DEBUG: true,
// Secure encryption salt. First 16 chars used for tripcode, mnemonics uses all 40
	SECURE_SALT: "LALALALALALALALALALALALALALALALALALALALA", /* [A-Za-z0-9]{40} */
// Relative path to serve websocket connections
	SOCKET_PATH: '/hana',
/*
 Absolute URL for client connections. Defaults to SOCKET_PATH. Only set this, if
 you are serving websockets from a different root address.
 */
	SOCKET_URL: null,
// Honour X-Forwarded-For HTTP headers for client IP determination
	TRUST_X_FORWARDED_FOR: true,
/*
 Use internal HTTP server to serve these resources.It is recommended to serve
 the www directory with a dedicated webserver, like nginx, and set MEDIAURL
 in imager/config.js to the served directory's address.
 */
	SERVE_STATIC_FILES: true,
	SERVE_IMAGES: true,
// Not preferred; use nginx (or other's) gzipping
	GZIP: true,
/*
 Enable usage of the websocket protocol (otherwise only emulation). Disabling
 this increases connection speed, if behind restrictive proxies.
 */
	USE_WEBSOCKETS: true,

	REDIS_PORT: 6379,
	redis_database: 0,
	READ_ONLY: false,

	BOARDS: ['moe', 'gar', 'meta', 'staff'],
	DEFAULT_BOARD: 'moe',
// Add links to the navigator menu to custom URLs. Also enables linking
// these in posts with `>>>/${board}/`.
	PSUEDO_BOARDS: [
		['g', 'https://google.com']
	],
// Only enable in-post links, without adding to the board navigation bar
	link_boards: [
		['4chan', 'http://www.4chan.org/']
	],
	STAFF_BOARD: 'staff',
// Boards with disabled moderation
	containment_boards: ['meta'],

// Language settings. You can easily map more. See ./lang/
	LANGS: ['en_GB', 'pt_BR', 'es_ES'],
	DEFAULT_LANG: 'en_GB',

// Thread creation cooldown for the same IP in seconds
	THREAD_THROTTLE: 60,
// Posting speed throttling settings
	SHORT_TERM_LIMIT: 2000,
	LONG_TERM_LIMIT: 2000*20*12,
	NEW_POST_WORTH: 50,
	IMAGE_WORTH: 50,

// Number of pages per board
	PAGES: {
		moe: 5,
		gar: 5,
		meta: 5,
		staff: 5
	},
// Number of posts per thread, after which the thread stops bumping to the
// top of the board
	BUMP_LIMIT: {
		moe: 1000,
		gar: 1000,
		meta: 1000,
		staff: 1000
	},
// Delete threads and their images, when they exceed the board's page limit
	PRUNE: false,

/*
 Doushio uses Mozilla's Persona system for staff authentication.
 Set login emails aliases, which will be used for logging, here.
 */
	staff: {
		admin: {'lalc@doushio.com': 'lalc'},
		moderator: {'mod@doushio.com': 'mod'},
		dj: {'dj@doushio.com': 'dj'},
		janitor: {'janitor@doushio.com': 'janny'}
	},
// You can log in/out by typing the following keyword in the email field
	LOGIN_KEYWORD: 'misaki',
// URL and domain of the website
	PERSONA_AUDIENCE: 'http://localhost:8000',
	LOGIN_SESSION_TIME: 60*60*24*14,

// r/a/dio integration (https://r-a-d.io)
	RADIO: false,
// Missle Launcher
	PYU: false
};

// Source the other config files
require('underscore').extend(module.exports,
	require('./imager'),
	require('./report')
);
