var async = require('async'),
    child_process = require('child_process'),
    config = require('./config'),
    imagerConfig = require('./imager/config'),
    fs = require('fs'),
    util = require('util');

var defines = {};
for (var k in config)
	defines[k] = JSON.stringify(config[k]);
for (var k in imagerConfig)
	defines[k] = JSON.stringify(imagerConfig[k]);

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

function lookup_config(key) {
	var val = config[key];
	if (val === undefined)
		val = imagerConfig[key];
	return val;
}

var config_re = /\bconfig\.(\w+)\b/;

async.forEachSeries(files, function (file, cb) {
	if (file.match(/^lib\//)) {
		process.stdout.write(fs.readFileSync(file));
		process.stdout.write('\n');
		return cb(null);
	}
	if (file.match(/^config\.js/))
		return cb("config.js shouldn't be in client");
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
		if (line.match(/^var\s+(config|common)\s*=\s*require.*$/))
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

		// XXX: risky
		line = line.replace(/\bcommon\.\b/g, '');

		while (true) {
			var m = line.match(config_re);
			if (!m)
				break;
			var cfg = lookup_config(m[1]);
			if (cfg === undefined) {
				console.error("No such config var " + m[1]);
				process.exit(1);
			}
			// Bleh
			if (cfg instanceof RegExp)
				cfg = cfg.toString();
			else
				cfg = JSON.stringify(cfg);
			line = line.replace(config_re, cfg);
		}
		for (var src in defines) {
			if (line.indexOf(src) < 0)
				continue;
			var regexp = new RegExp('(?:DEFINES\.)?\\b' + src
					+ '\\b', 'g');
			line = line.replace(regexp, defines[src]);
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
});
