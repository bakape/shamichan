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
	var arg = process.argv[2];
	if (arg == '--client-version') {
		var deps = require('./deps');
		get_version(deps.CLIENT_DEPS, function (err, version) {
			if (err)
				throw err;
			else
				console.log(version);
		});
	}
	else if (arg.match(/^(CLIENT|SERVER)_DEPS/)) {
		console.log(require('./deps')[arg].join(' '));
	}
	else {
		var config = require('./config');
		if (!(arg in config))
			throw "No such config value " + arg;
		console.log(config[arg]);
	}
}
