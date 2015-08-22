/*
Parse process arguments. Usefull for running several servers in the same
 root directory.
 */
'use strict';

const config = require('../config'),
    minimist = require('minimist'),
    path = require('path');

function usage() {
	process.stderr.write(
`Usage: node server/server.js
    -h || --help      Display this help text
    --host <host>     Override server listening host
    --port <port>     Override server listening port.
                      <port> can aslso be Unix domain socket path.
    --pid <pid path>  Override pid file path
    --debug           Force debug mode
`);
	process.exit(1);
}

function parse_args() {
	const argv = minimist(process.argv.slice(2));

	if ('h' in argv || 'help' in argv)
		return usage();

	if (argv.port)
		config.LISTEN_PORT = argv.port;
	if (argv.host)
		config.LISTEN_HOST = argv.host;
	if (argv.debug) 
		config.DEBUG = true;
	config.PID_FILE = argv.pid || path.join(__dirname, '.server.pid');
}
exports.parse_args = parse_args;
