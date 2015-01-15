var async = require('async'),
    child_process = require('child_process'),
    config = require('./config'),
    imagerConfig = require('./imager/config'),
    reportConfig = require('./report/config'),
    fs = require('fs'),
    util = require('util');

function make_client(inputs, out, cb) {

var defines = {};
for (var k in config)
	defines[k] = JSON.stringify(config[k]);
for (var k in imagerConfig)
	defines[k] = JSON.stringify(imagerConfig[k]);
for (var k in reportConfig)
	defines[k] = JSON.stringify(reportConfig[k]);

// UGH
var configDictLookup = {
	config: config,
	imagerConfig: imagerConfig,
	reportConfig: reportConfig,
};

function lookup_config(dictName, key) {
	var dict = configDictLookup[dictName];
	if (key.indexOf('SECURE') >= 0 || key.indexOf('PRIVATE') >= 0)
		throw new Error("Refusing " + key + " in client code!");
	return dict[key];
}

var config_re = /\b(\w+onfig)\.(\w+)\b/;

function convert(file, cb) {
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
			var cfg = lookup_config(m[1], m[2]);
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

function make_minified(files, out, cb) {

	// would be nice if uglify streamed...
	require('tmp').file({
		postfix: '.gen.js',
	},
	function (err, tmp, fd) {
		if (err) return cb(err);
		var out = fs.createWriteStream(null, {fd: fd});
		out.once('error', cb);
		make_client(files, out, function (err) {
			if (err)
				return cb(err);
			out.end(function () {
				minify(tmp);
			});
		});
	});

	function minify(file) {
		var UglifyJS = require('uglify-js');
		var ugly;
		try {
			ugly = UglifyJS.minify(file, {
				mangle: false,
			});
		}
		catch (e) {
			return cb(e);
		}
		out.write(ugly.code, cb);
	}
};

exports.make_minified = make_minified;

function make_maybe_minified(files, out, cb) {
	if (config.DEBUG)
		make_client(files, out, cb);
	else
		make_minified(files, out, cb);
}

exports.make_maybe_minified = make_maybe_minified;

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

	make_maybe_minified(files, process.stdout, function (err) {
		if (err) throw err;
	});
}
