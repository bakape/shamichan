var config = require('./config'),
    fs = require('fs'),
    util = require('util');

var defines = {};
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

for (var i = 0; i < files.length; i++) {
	var lines = fs.readFileSync(files[i], 'UTF-8').split('\n');
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
		if (m) {
			line = 'var ' + m[1] + ' = ' + m[2];
		}

		for (var src in defines) {
			if (line.indexOf(src) < 0)
				continue;
			var regexp = new RegExp('\\b' + src + '\\b');
			line = line.replace(regexp, defines[src]);
		}
		line = line.replace('DEFINES.', '');
		line = line.replace('exports.', '');
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
		console.log(line);
	}
}
