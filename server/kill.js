#!/usr/bin/env node
var config = require('../config'),
    path = require('path');

var cfg = config.DAEMON;
if (cfg) {
	var lock = path.join(cfg.PID_PATH, 'server.pid');
	require('daemon').kill(lock, function (err) {
		if (err)
			throw err;
	});
}
else {
	/* non-daemon version for hot reloads */
	var file = path.join(path.dirname(module.filename), '.server.pid');
	require('fs').readFile(file, function (err, pid) {
		pid = parseInt(pid, 10);
		if (err || !pid)
			return console.warn('No pid.');
		require('child_process').exec('kill -HUP ' + pid,
					function (err) {
			if (err) throw err;
			if (process.argv.indexOf('--silent') < 2)
				console.log('Sent HUP.');
		});
	});
}
