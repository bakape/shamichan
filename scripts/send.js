/*
 Send a mesage to all clients with push notifications
 */

'use strict';

const argv = require('minimist')(process.argv.slice(2)),
	config = require('../config'),
	redis = require('redis').createClient(config.REDIS_PORT);

if ('h' in argv || 'help' in argv || !argv._.length)
	usage();

redis.select(config.redis_database || 0);

const msg = JSON.stringify(argv._);
redis.publish('push', msg, function (err) {
	if (err)
		return console.error('Error: Failed to send: ', msg);
	console.log('Sent: ', msg);
	process.exit();
});

function usage() {
	process.stderr.write(
`Sends a message through push notifications to all active clients
Usage: node scripts/send.js <msg>
  -h --help Displays this message
  <msg> contents of the array to be sent in complience with the websocket API
    Example: node scripts/send 0 39 'notification test'
`
	);
	process.exit(1);
}
