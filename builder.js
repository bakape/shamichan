var _ = require('underscore'),
    config = require('./config'),
    deps = require('./deps'),
    fs = require('fs'),
    child_process = require('child_process');

if (config.DAEMON)
	throw "Can't run dev server in daemon mode.";

var server;
var start_server = _.debounce(function () {
	if (server)
		server.kill('SIGTERM');
	server = child_process.spawn('node', ['server/server.js']);
	server.stdout.pipe(process.stdout);
	server.stderr.pipe(process.stderr);
}, 2000);

function rebuildClient(cb) {
	var make = child_process.spawn('make', ['client']);
	make.stdout.pipe(process.stdout);
	make.stderr.pipe(process.stderr);
	make.on('error', function(err) {
		console.error(err);
	});
	make.on('exit', cb);
}

var reload_state = _.debounce(function () {
	rebuildClient(function() {
		if (server)
			server.kill('SIGHUP');
	});
}, 2000);

deps.SERVER_DEPS.forEach(monitor.bind(null, start_server));
deps.SERVER_STATE.forEach(monitor.bind(null, reload_state));
deps.CLIENT_DEPS.forEach(monitor.bind(null, reload_state));
deps.ALPHA.forEach(monitor.bind(null, reload_state));

function monitor(func, dep) {
	var mtime = new Date;
	fs.watchFile(dep, {interval: 500, persistent: true}, function (event) {
		if (event.mtime > mtime) {
			func();
			mtime = event.mtime;
		}
	});
}

rebuildClient(start_server);
