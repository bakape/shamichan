var config = require('../config'),
    minimist = require('minimist');

function usage() {
	process.stderr.write(
	  "Usage: node server/server.js\n"
	+ "       --host <host> --port <port>\n"
	+ "\n"
	+ "<port> can also be a unix domain socket path.\n"
	);
	process.exit(1);
}

exports.parse_args = function () {
	var argv = minimist(process.argv.slice(2));

	if ('h' in argv || 'help' in argv)
		return usage();

	if (argv.port)
		config.LISTEN_PORT = argv.port;
	if (argv.host)
		config.LISTEN_HOST = argv.host;
};
