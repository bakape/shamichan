/*
Self-reloading and rebuilding development server
 */
'use strict';

const _ = require('underscore'),
	config = require('./config'),
	deps = require('./deps'),
	child_process = require('child_process'),
	watch = require('node-watch');

let server;
const start_server = _.debounce(function() {
	if (server)
		server.kill('SIGTERM');
	server = child_process.spawn('node', ['index']);
	server.stdout.pipe(process.stdout);
	server.stderr.pipe(process.stderr);
}, 2000);

function build(args, cb) {
	var cp = child_process.spawn('./node_modules/.bin/gulp', args);
	cp.stdout.pipe(process.stdout);
	cp.stderr.pipe(process.stderr);
	cp.on('error', function(err) {
		console.error(err);
	});
	cp.on('exit', cb);
}

const buildAll = build.bind(null, [
	'client',
	'vendor',
	'css',
	'mod',
	'lang',
	'legacy'
]);
const reload_state = _.debounce(function() {
	if (server)
		server.kill('SIGHUP');
}, 2000);
const fullRestart = _.debounce(function() {
	buildAll(start_server);
}, 5000);

const serverExclude = new RegExp(
	String.raw`\.pid$|\.socket|hot.js$|`
		+ config.MEDIA_DIRS.tmp.replace('/', '\\/')
);
watch(deps.state, reload_state);
watch(deps.server, function(file) {
	/*
	 PID file is generated each start and `hot.js` should only triigger a
	 reaload.
	 */
	if (!serverExclude.test(file))
		start_server();
});
watch('common', fullRestart);
watch('lang', fullRestart);
watch('gulpfile.js', function() {
	buildAll(reload_state);
});
['mod', 'client', 'css'].forEach(function(task) {
	watch(deps[task], _.debounce(function() {
		build([task], reload_state);
	}), 5000);
});

// Initial start
buildAll(start_server);
