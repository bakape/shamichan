var fs = require('fs');

var defines = {};

for (var i = 2; i < process.argv.length; i++) {
	var lines = fs.readFileSync(process.argv[i], 'UTF-8').split('\n');
	for (var j = 0; j < lines.length; j++) {
		var line = lines[j];
		m = line.match(/^exports\.(\w+)\s*=\s*([^{]+);$/);
		if (m) {
			if (m[1] !== m[2])
				defines[m[1]] = m[2];
			continue;
		}
		m = line.match(/^exports\.(\w+)\s*=\s*(.*)$/);
		if (m)
			line = 'var ' + m[1] + ' = ' + m[2];
		for (var src in defines) {
			if (line.indexOf(src) < 0)
				continue;
			var regexp = new RegExp('\\b' + src + '\\b');
			line = line.replace(regexp, defines[src]);
		}
		console.log(line);
	}
}
