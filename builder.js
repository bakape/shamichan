var _ = require('underscore'),
	config = require('./config'),
	deps = require('./deps'),
	child_process = require('child_process'),
	watch = require('node-watch');

if (config.DAEMON)
	throw "Can't run dev server in daemon mode.";

var server;
var start_server = _.debounce(function() {
	if (server)
		server.kill('SIGTERM');
	server = child_process.spawn('node', ['server/server.js']);
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

var reload_state = _.debounce(function() {
	if (server)
		server.kill('SIGHUP');
}, 2000);

['client', 'css', 'mod'].forEach(function(task) {
	watch(deps[task], _.debounce(function() {
		build([task], reload_state);
	}), 5000);
});


watch(deps.state, reload_state);

const serverExclude = new RegExp('\\.pid$|hot.js$|admin\\/client.js$|'
	+ config.MEDIA_DIRS.tmp.replace('/', '\\/'));
watch(deps.server, function(file) {
	/*
	 PID file is generated each start and `hot.js` should only triigger a
	 reaload.
	 */
	if (!serverExclude.test(file))
		start_server();
});

// Initial start
build(['client', 'css', 'mod'], start_server);
