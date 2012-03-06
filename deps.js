var config = require('./config');

var minJs = config.DEBUG ? '.js' : '.min.js';

exports.CLIENT_DEPS = [
	'lib/underscore' + minJs,
	'lib/backbone' + minJs,
	'lib/socket.io' + minJs,
	'lib/oninput' + minJs,
	'common.js',
	'client/client.js',
];

exports.SERVER_DEPS = [
	'common.js',
	'config.js',
	'db.js',
	'get.js',
	'lib/underscore.js',
	'server/games.js',
	'server/perceptual.c',
	'server/pix.js',
	'server/server.js',
	'server/state.js',
	'server/twitter.js',
	'server/tripcode.cc',
];

// Changes to these only require a state.js reload
exports.SERVER_STATE = [
	'client/mod.js',
	'hot.js',
	'tmpl/filter.html',
	'tmpl/index.html',
];
