var config = require('./config');

var minJs = config.DEBUG ? '.js' : '.min.js';

exports.CLIENT_DEPS = [
	'lib/underscore' + minJs,
	'lib/backbone' + minJs,
	'lib/oninput' + minJs,
	'common.js',
	'client/options.js',
	'client/client.js',
	'client/postform.js',
	'client/conn.js',
	'client/amusement.js',
];

exports.SERVER_DEPS = [
	'common.js',
	'config.js',
	'db.js',
	'deps.js',
	'get.js',
	'hooks.js',
	'lib/underscore.js',
	'server/amusement.js',
	'server/caps.js',
	'server/panel.js',
	'server/perceptual.c',
	'server/pix.js',
	'server/server.js',
	'server/state.js',
	'server/twitter.js',
	'server/tripcode.cc',
	'server/web.js',
];

// Changes to these only require a state.js reload
exports.SERVER_STATE = [
	'client/mod.js',
	'hot.js',
	'tmpl/filter.html',
	'tmpl/index.html',
];
