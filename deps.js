var config = require('./config');

exports.SERVER_DEPS = [
	'admin/common.js',
	'admin/index.js',
	'admin/panel.js',
	'autoJoe/server.js',
	'common.js',
	'config.js',
	'db.js',
	'deps.js',
	'etc.js',
	'hooks.js',
	'tail.js',
	'curfew/server.js',
	'imager/config.js',
	'imager/daemon.js',
	'imager/db.js',
	'imager/index.js',
	'imager/jobs.js',
	'report/config.js',
	'report/server.js',
	'server/amusement.js',
	'server/api.js',
	'server/caps.js',
	'server/msgcheck.js',
	'server/okyaku.js',
	'server/opts.js',
	'server/perceptual.c',
	'server/persona.js',
	'server/render.js',
	'server/server.js',
	'server/state.js',
	'server/web.js',
	'time/server.js',
	'tripcode/tripcode.cc',
];

/* Changes to the below only require a state.js reload */

exports.VENDOR_DEPS = [
	'./node_modules/jquery/dist/jquery.js',
	'./lib/yepnope.js',
	'./node_modules/underscore/underscore.js',
	'./node_modules/backbone/backbone.js',
	'./lib/oninput.js',
	'./node_modules/jquery.cookie/jquery.cookie.js',
	'./lib/pixastic.blurfast.min.js',
];

exports.CLIENT_DEPS = [
	'common.js',
	'client/init.js',
	'client/memory.js',
	'client/imager.js',
	'client/models.js',
	'client/extract.js',
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
	'client/banner.js',
	'client/mobile.js',
];

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
