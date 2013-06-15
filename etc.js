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
	fs.mkdir(dir, function (err) {
		cb(err && err.code == 'EEXIST' ? null : err);
	});
};
