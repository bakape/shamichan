exports.SERVER_DEPS = [
	'admin/common.js',
	'admin/panel.js',
	'common/imports.js',
	'common/index.js',
	'common/oneesama.js',
	'common/options.js',
	'common/util.js',
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
	'radio/server.js',
	'report/config.js',
	'report/server.js',
	'server/amusement.js',
	'server/api.js',
	'server/archive.js',
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
	'tripcode/tripcode.cc'
];

// Changes to the below only require a state.js reload
exports.SERVER_STATE = [
	'admin/client.js',
	'hot.js',
	'tmpl/curfew.html',
	'tmpl/index.html',
	'tmpl/login.html',
	'tmpl/redirect.html'
];

// Run `make client` and reload state.js to apply changes to the below
exports.MOD_CLIENT_DEPS = [
	'admin/index.js',
	'admin/client.js'
];

// Only used for monitoring by builder.js
exports.ALPHA = [
	'gulpfile.js',
	'common/imports.js',
	'common/index.js',
	'common/oneesama.js',
	'common/options.js',
	'common/util.js',
	'client/amusement.js',
	'client/background.js',
	'client/banner.js',
	'client/client.js',
	'client/connection.js',
	'client/extract.js',
	'client/history.js',
	'client/hover.js',
	'client/main.js',
	'client/memory.js',
	'client/mobile.js',
	'client/options',
	'client/posts',
	'client/scroll.js',
	'client/state.js',
	'client/time.js',
	'client/options/index.js',
	'client/posts/article.js',
	'client/posts/embed.js',
	'client/posts/identity.js',
	'client/posts/index.js',
	'client/posts/imager.js',
	'client/posts/index.js',
	'client/posts/models.js',
	'client/posts/nonce.js',
	'client/posts/posting.js',
	'client/posts/section.js'
];
