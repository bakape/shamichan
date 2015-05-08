var config = require('../config'),
    minimist = require('minimist'),
    path = require('path');

function usage() {
	process.stderr.write(
	  "Usage: node server/server.js\n"
	+ "       --host <host> --port <port>\n"
	+ "       --pid <pid file location>\n"
	+ "\n"
	+ "<port> can also be a unix domain socket path.\n"
	);
	process.exit(1);
}

function parse_args () {
	var argv = minimist(process.argv.slice(2));

	if ('h' in argv || 'help' in argv)
		return usage();

	if (argv.port)
		config.LISTEN_PORT = argv.port;
	if (argv.host)
		config.LISTEN_HOST = argv.host;
	if (argv.pid)
		config.PID_FILE = argv.pid;
}
exports.parse_args = parse_args;

function load_defaults () {
	if (!config.PID_FILE)
		config.PID_FILE = path.join(__dirname, '.server.pid');
}
exports.load_defaults = load_defaults;
