var _ = require('./lib/underscore'),
    config = require('./config'),
    deps = require('./deps'),
    fs = require('fs'),
    child_process = require('child_process');

var server_deps = deps.SERVER_DEPS.slice();
var client_deps = deps.CLIENT_DEPS.slice();

var server;
var start_server = _.debounce(function () {
	if (server)
		server.kill('SIGTERM');
	server = child_process.spawn('node', ['server/server.js']);
	server.stdout.pipe(process.stdout);
	server.stderr.pipe(process.stderr);
}, 500);

var build_client = _.debounce(function () {
	var make = child_process.execFile('make', ['-s', '-q', 'client']);
	make.once('exit', function (code) {
		if (!code)
			return;
		console.log('make client');
		var make = child_process.execFile('make', ['-s', 'client']);
		make.stdout.pipe(process.stdout);
		make.stderr.pipe(process.stderr);
	});
}, 500);

server_deps.forEach(monitor.bind(null, start_server));
client_deps.forEach(monitor.bind(null, build_client));

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
build_client();
