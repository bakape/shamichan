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
	'lib/underscore.js',
	'admin.js',
	'common.js',
	'config.js',
	'db.js',
	'games.js',
	'get.js',
	'pix.js',
	'server.js',
	'state.js',
	'twitter.js',
	'tripcode.cc',
	'filter.html',
	'index.html',
];
