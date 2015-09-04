/*
 Websocket handler module
 */

var _ = require('underscore'),
	async = require('async'),
	caps = require('./caps'),
    common = require('../common/index'),
    events = require('events'),
    Muggle = require('../util/etc').Muggle,
    STATE = require('./state'),
    util = require('util'),
    winston = require('winston');

var dispatcher = exports.dispatcher = {};

function Okyaku(socket, ip) {
	events.EventEmitter.call(this);

	this.socket = socket;
	this.ident = caps.lookup_ident(ip);
	this.watching = {};
	this.ip = ip;

	var clients = STATE.clientsByIP[ip];
	if (clients)
		clients.push(this);
	else
		clients = STATE.clientsByIP[ip] = [this];
	STATE.emitter.emit('change:clientsByIP', ip, clients);
}
util.inherits(Okyaku, events.EventEmitter);
exports.Okyaku = Okyaku;

var OK = Okyaku.prototype;

OK.send = function (msg) {
	this.socket.write(JSON.stringify([msg]));
};

OK.on_update = function (op, kind, msg) {
	// Special cases for operations that overwrite a client's state
	const {post} = this;
	if (post && kind == common.DELETE_POSTS) {
		const num = JSON.parse(msg)[0].slice(2)[0];
		if (num === post.num || num === post.op)
			this.post = null;
	}
	this.socket.write(msg);
};

OK.on_thread_sink = function (thread, err) {
	/* TODO */
	winston.error(thread + ' sank: ' + err);
};

OK.on_message = function (data) {
	if (this.ident.ban)
		return;
	var msg;
	try { msg = JSON.parse(data); }
	catch (e) {}
	var type = common.INVALID;
	if (msg) {
		if (this.post && typeof msg == 'string')
			type = common.UPDATE_POST;
		else if (msg.constructor == Array)
			type = msg.shift();
	}
	if (!this.synced && type != common.SYNCHRONIZE)
		type = common.INVALID;
	var func = dispatcher[type];
	if (!func || !func(msg, this)) {
		this.kotowaru(Muggle("Bad protocol",
			new Error("Invalid message: " + JSON.stringify(data))));
	}
};

var ip_expiries = {};

OK.on_close = function () {
	var ip = this.ip;
	var clientList = STATE.clientsByIP[ip];
	if (clientList) {
		var i = clientList.indexOf(this);
		if (i >= 0) {
			clientList.splice(i, 1);
			STATE.emitter.emit('change:clientsByIP',ip,clientList);
		}
		if (!clientList.length) {
			// Expire this list after a short delay
			if (ip_expiries[ip])
				clearTimeout(ip_expiries[ip]);
			ip_expiries[ip] = setTimeout(function () {
				var list = STATE.clientsByIP[ip];
				if (list && list.length === 0)
					delete STATE.clientsByIP[ip];
				delete ip_expiries[ip];
			}, 5000);
		}
	}

	if (this.id) {
		delete STATE.clients[this.id];
		this.id = null;
	}
	this.synced = false;
	var db = this.db;
	if (db) {
		db.kikanai();
		if (this.post)
			this.finish_post(function () {
				db.disconnect();
			});
		else
			db.disconnect();
	}

	this.emit('close');
};

OK.kotowaru = function (error) {
	if (this.blackhole)
		return;
	var msg = 'Server error.';
	if (error instanceof Muggle) {
		msg = error.most_precise_error_message();
		error = error.deepest_reason();
	}
	winston.error(`Error by ${JSON.stringify(this.ident)}: ${error || msg}`);
	this.send([0, common.INVALID, msg]);
	this.synced = false;
};

OK.finish_post = function (callback) {
	/* TODO: Should we check this.uploading? */
	var self = this;
	this.db.finish_post(this.post, function (err) {
		if (err)
			callback(err);
		else {
			if (self.post) {
				self.last_num = self.post.num;
				self.post = null;
			}
			callback(null);
		}
	});
};

function scan_client_caps() {
	const clients = STATE.clientsByIP;
	for (let ip in clients) {
		const ident = caps.lookup_ident(ip);
		if (!ident.ban)
			continue;

		// The length of the array changes, so make a shallow copy
		for (let okyaku of clients[ip].slice()) {
			okyaku.ident.ban = true;
			try {
				okyaku.socket.close();
			}
			catch (e) {
				// Already closed. Whatever.
			}
		}
	}
}
exports.scan_client_caps = scan_client_caps;

// Push message to all clients
function push(msg){
	for (let client of _.values(STATE.clients)) {
		try {
			client.send(msg);
		}
		catch(e){
			// Client died, but we don't care
		}
	}
}
exports.push = push;
