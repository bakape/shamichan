var config = require('./config');

var minJs = config.DEBUG ? '.js' : '.min.js';

exports.CLIENT_DEPS = [
	'lib/yepnope' + minJs,
	'lib/underscore' + minJs,
	'lib/backbone' + minJs,
	'lib/oninput' + minJs,
	'lib/jquery.cookie' + minJs,
	'common.js',
	'client/init.js',
	'client/models.js',
	'client/options.js',
	'client/scroll.js',
	'client/client.js',
	'client/posting.js',
	'client/menu.js',
	'client/conn.js',
	'client/unread.js',
	'client/drop.js',
	'client/hide.js',
	'client/hover.js',
	'client/amusement.js',
	'client/embed.js',
	'curfew/client.js',
	'voice/client.js',
];

exports.SERVER_DEPS = [
	'authcommon.js',
	'common.js',
	'config.js',
	'db.js',
	'deps.js',
	'get.js',
	'hooks.js',
	'muggle.js',
	'tail.js',
	'curfew/server.js',
	'lib/underscore.js',
	'imager/config.js',
	'imager/daemon.js',
	'imager/db.js',
	'imager/index.js',
	'imager/jobs.js',
	'server/amusement.js',
	'server/caps.js',
	'server/msgcheck.js',
	'server/okyaku.js',
	'server/panel.js',
	'server/perceptual.c',
	'server/persona.js',
	'server/server.js',
	'server/state.js',
	'server/web.js',
	'tripcode/tripcode.cc',
	'voice/server.js',
];

// Changes to these only require a state.js reload
exports.SERVER_STATE = [
	'client/admin.js',
	'hot.js',
	'tmpl/alookup.html',
	'tmpl/curfew.html',
	'tmpl/filter.html',
	'tmpl/index.html',
	'tmpl/login.html',
	'tmpl/redirect.html',
];

exports.MOD_CLIENT_DEPS = [
	'authcommon.js',
	'client/admin.js',
];
