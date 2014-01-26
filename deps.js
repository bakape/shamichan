var config = require('./config');

var minJs = config.DEBUG ? '.js' : '.min.js';

exports.VENDOR_DEPS = [
	'lib/yepnope' + minJs,
	'lib/underscore' + minJs,
	'lib/backbone' + minJs,
	'lib/oninput' + minJs,
	'lib/jquery.cookie' + minJs,
];

exports.CLIENT_DEPS = [
	'common.js',
	'client/init.js',
	'client/memory.js',
	'client/models.js',
	'client/options.js',
	'client/scroll.js',
	'client/client.js',
	'client/posting.js',
	'client/menu.js',
	'client/conn.js',
	'client/notify.js',
	'client/drop.js',
	'client/hide.js',
	'client/hover.js',
	'client/amusement.js',
	'client/embed.js',
	'client/gravitas.js',
	'curfew/client.js',
	'report/client.js',
	'time/client.js',
	'voice/client.js',
];

exports.SERVER_DEPS = [
	'admin/common.js',
	'admin/index.js',
	'admin/panel.js',
	'common.js',
	'config.js',
	'db.js',
	'deps.js',
	'etc.js',
	'hooks.js',
	'make_client.js',
	'pipeline.js',
	'tail.js',
	'curfew/server.js',
	'lib/underscore.js',
	'imager/config.js',
	'imager/daemon.js',
	'imager/db.js',
	'imager/index.js',
	'imager/jobs.js',
	'report/config.js',
	'report/server.js',
	'server/amusement.js',
	'server/caps.js',
	'server/msgcheck.js',
	'server/okyaku.js',
	'server/perceptual.c',
	'server/persona.js',
	'server/render.js',
	'server/server.js',
	'server/state.js',
	'server/web.js',
	'tripcode/tripcode.cc',
	'voice/server.js',
];

// Changes to these only require a state.js reload
exports.SERVER_STATE = [
	'admin/client.js',
	'hot.js',
	'tmpl/alookup.html',
	'tmpl/curfew.html',
	'tmpl/filter.html',
	'tmpl/index.html',
	'tmpl/login.html',
	'tmpl/redirect.html',
];

exports.MOD_CLIENT_DEPS = [
	'admin/common.js',
	'admin/client.js',
];
