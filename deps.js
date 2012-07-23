var config = require('./config');

var minJs = config.DEBUG ? '.js' : '.min.js';

exports.CLIENT_DEPS = [
	'lib/underscore' + minJs,
	'lib/backbone' + minJs,
	'lib/oninput' + minJs,
	'common.js',
	'client/init.js',
	'client/models.js',
	'client/options.js',
	'client/scroll.js',
	'client/client.js',
	'client/posting.js',
	'client/menu.js',
	'client/conn.js',
	'client/drop.js',
	'client/amusement.js',
	'client/youtube.js',
];

exports.SERVER_DEPS = [
	'authcommon.js',
	'common.js',
	'config.js',
	'db.js',
	'deps.js',
	'get.js',
	'hooks.js',
	'lib/underscore.js',
	'server/amusement.js',
	'server/caps.js',
	'server/msgcheck.js',
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
	'client/admin.js',
	'hot.js',
	'tmpl/curfew.html',
	'tmpl/filter.html',
	'tmpl/index.html',
];

exports.MOD_CLIENT_DEPS = [
	'authcommon.js',
	'client/admin.js',
];
