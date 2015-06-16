/*
 Send a mesage to all clients with push notifications
 */

'use strict';

let argv = require('minimist')(process.argv.slice(2)),
	net = require('net'),
	path = require('path');

function parseArgs() {
	if ('h' in argv || 'help' in argv)
		return usage();
	if (!argv._.length)
		return usage();
	let socket = net.createConnection('./server/.socket');
	socket.once('connect', function() {
		const msg = JSON.stringify(argv._);
		if (socket.write(msg))
			console.log('Sent: ', msg);
		else
			console.error('Error: Failed to send: ', msg);
		process.exit(1);
	})
}

function usage() {
	process.stderr.write(
`Sends a message through push notifications to all active clients
Usage: node scripts/send.js <msg>
  -h --help Displays this message
  <msg> contents of the array to be sent in complience with the websocket API
    Example: 0 39 'notification test'
`
	);
}

parseArgs();
