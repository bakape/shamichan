var config = require('./config');

function get_version(deps, callback) {
	require('child_process').exec('git log -1 --format=%h '+deps.join(' '),
			function (err, stdout, stderr) {
		if (err)
			callback(err);
		else
			callback(null, stdout.trim());
	});
}
exports.get_version = get_version;

if (process.argv[1] == __filename) {
	if (process.argv.length != 3) {
		console.error("Specify a config key or --client-version.");
		process.exit(-1);
	}
	if (process.argv[2] == '--client-version') {
		get_version(config.CLIENT_DEPS, function (err, version) {
			if (err)
				throw err;
			else
				console.log(version);
		});
	}
	else {
		var key = process.argv[2];
		if (!(key in config))
			throw "No such config value " + key;
		var val = config[key];
		console.log((val && val.join) ? val.join(' ') : val);
	}
}
