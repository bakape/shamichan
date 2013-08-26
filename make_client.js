var async = require('async'),
    child_process = require('child_process'),
    config = require('./config'),
    imagerConfig = require('./imager/config'),
    fs = require('fs'),
    util = require('util');

function make_client(inputs, out, cb) {

var defines = {};
for (var k in config)
	defines[k] = JSON.stringify(config[k]);
for (var k in imagerConfig)
	defines[k] = JSON.stringify(imagerConfig[k]);

function lookup_config(key) {
	var val = config[key];
	if (val === undefined)
		val = imagerConfig[key];
	return val;
}

var config_re = /\b(\w+onfig)\.(\w+)\b/;

function convert(file, cb) {
	if (/^lib\//.test(file)) {
		fs.readFile(file, 'UTF-8', function (err, lib) {
			if (err)
				return cb(err);
			out.write(lib);
			out.write('\n');
			cb(null);
		});
		return;
	}
	if (/^config\.js/.test(file))
		return cb("config.js shouldn't be in client");

fs.readFile(file, 'UTF-8', function (err, fullFile) {
	if (err)
		return cb(err);

	var lines = fullFile.split('\n');
	var waitForDrain = false;
	for (var j = 0; j < lines.length; j++) {
		var line = lines[j];
		if (/^var\s+DEFINES\s*=\s*exports\s*;\s*$/.test(line))
			continue;
		if (/^var\s+(\w+onfig|common|_)\s*=\s*require.*$/.test(line))
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
			var dict = m[1] == 'config' ? config : imagerConfig;
			var cfg = dict[m[2]];
			if (cfg === undefined) {
				return cb("No such "+m[1]+" var "+m[2]);
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
		waitForDrain = !out.write(line+'\n', 'UTF-8');
	}
	if (waitForDrain)
		out.once('drain', function () { cb(null); });
	else
		cb(null);

}); // readFile
}

	// kick off
	async.eachSeries(inputs, convert, cb);
};

exports.make_client = make_client;

if (require.main === module) {
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

	make_client(files, out, function (err) {
		if (err) throw err;
	});
}
