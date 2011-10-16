var async = require('async'),
    child_process = require('child_process'),
    config = require('./config'),
    fs = require('fs'),
    util = require('util');

var defines = {};
for (var k in config)
	defines[k] = JSON.stringify(config[k]);
var files = [];
for (var i = 2; i < process.argv.length; i++) {
	var arg = process.argv[i];
	if (arg[0] != '-') {
		files.push(arg);
		continue;
	}
	else {
		util.error('Unrecognized option ' + arg);
		process.exit(1);
	}
}

var config_re = /\$\*\$(\w+)\$\*\$/;

async.forEachSeries(files, function (file, cb) {
	if (file.match(/^lib\//)) {
		process.stdout.write(fs.readFileSync(file));
		return cb(null);
	}
	var lines = fs.readFileSync(file, 'UTF-8').split('\n');
	var out;
	if (config.DEBUG) {
		out = process.stdout;
	}
	else {
		var jsmin = child_process.spawn('./jsmin');
		jsmin.stdout.pipe(process.stdout, {end: false});
		jsmin.stderr.pipe(process.stderr, {end: false});
		jsmin.on('exit', cb);
		jsmin.stdin.on('error', cb);
		jsmin.stdout.on('error', cb);
		out = jsmin.stdin;
	}
	for (var j = 0; j < lines.length; j++) {
		var line = lines[j];
		if (line.match(/^var\s+DEFINES\s*=\s*exports\s*;\s*$/))
			continue;
		m = line.match(/^DEFINES\.(\w+)\s*=\s*(.+);$/);
		if (m) {
			defines[m[1]] = m[2];
			continue;
		}
		m = line.match(/^exports\.(\w+)\s*=\s*(\w+)\s*;\s*$/);
		if (m && m[1] == m[2])
			continue;
		m = line.match(/^exports\.(\w+)\s*=\s*(.*)$/);
		if (m)
			line = 'var ' + m[1] + ' = ' + m[2];

		for (var src in defines) {
			if (line.indexOf(src) < 0)
				continue;
			var regexp = new RegExp('\\b' + src + '\\b');
			line = line.replace(regexp, defines[src]);
		}
		line = line.replace('DEFINES.', '');
		while (true) {
			var m = line.match(config_re);
			if (!m)
				break;
			var cfg = config[m[1]];
			if (cfg === undefined) {
				console.error("No such config var " + m[1]);
				process.exit(1);
			}
			line = line.replace(config_re, cfg);
		}
		out.write(line+'\n', 'UTF-8');
	}
	if (config.DEBUG)
		cb(null);
	else
		out.end();
}, function (err) {
	if (err)
		throw err;
	process.stdout.end();
});
