/*
 Initialises the SockJS server
 */

'use strict';

let config = require('../../config'),
	fs = require('fs'),
	okyaku = require('../okyaku'),
	util = require('./util'),
	winston = require('winston');

const sockjsPath = 'js/' + fs
	.readFileSync('tmpl/index.html', 'UTF-8')
	.match(/sockjs-[\d.]+(?:\.min)?\.js/)[0];

var sockJs = require('sockjs').createServer({
	sockjs_url: config.MEDIA_URL + sockjsPath,
	prefix: config.SOCKET_PATH,
	jsessionid: false,
	log: sockjs_log,
	websocket: config.USE_WEBSOCKETS
});

function sockjs_log(sev, message) {
	if (sev === 'info')
		winston.verbose(message);
	else if (sev === 'error')
		winston.error(message);
}

sockJs.on('connection', function(socket) {
	let ip = socket.remoteAddress;
	if (config.TRUST_X_FORWARDED_FOR) {
		const ff = util.parse_forwarded_for(socket.headers['x-forwarded-for']);
		if (ff)
			ip = ff;
	}
	let client = new okyaku.Okyaku(socket, ip);
	socket.on('data', client.on_message.bind(client));
	socket.on('close', client.on_close.bind(client));
});

exports.start = function(server) {
	server.on('upgrade', function(req, resp) {
		resp.end();
	});
	sockJs.installHandlers(server);
};
