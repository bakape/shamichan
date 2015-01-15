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
}, 500);

var reload_state = _.debounce(function () {
	if (server)
		server.kill('SIGHUP');
}, 500);

deps.SERVER_DEPS.forEach(monitor.bind(null, start_server));
deps.SERVER_STATE.forEach(monitor.bind(null, reload_state));
deps.CLIENT_DEPS.forEach(monitor.bind(null, reload_state));

function monitor(func, dep) {
	var mtime = new Date;
	fs.watchFile(dep, {interval: 500, persistent: true}, function (event) {
		if (event.mtime > mtime) {
			func();
			mtime = event.mtime;
		}
	});
}

start_server();
