var child_process = require('child_process'),
    fs = require('fs'),
    Muggle = require('./muggle').Muggle;

exports.movex = function (src, dest, callback) {
	child_process.execFile('/bin/mv', ['-n', '--', src, dest],
				function (err, stdout, stderr) {
		if (err)
			callback(Muggle("Couldn't move file into place.",
					stderr || err));
		else
			callback(null);
	});
};

exports.checked_mkdir = function (dir, cb) {
	fs.stat(dir, function (err, info) {
		var make = false;
		if (err) {
			if (err.code == 'ENOENT')
				make = true;
			else
				return cb(err);
		}
		else if (!info.isDirectory())
			return cb(dir + " is not a directory");
		if (make)
			fs.mkdir(dir, cb);
		else
			cb(null);
	});
};
