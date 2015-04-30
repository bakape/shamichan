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
	if (this.post && kind == common.DELETE_POSTS) {
		var nums = JSON.parse(msg)[0].slice(2);
		if (nums.indexOf(this.post.num) >= 0)
			this.post = null;
	}
	else if (this.post && kind == common.DELETE_THREAD) {
		if (this.post.num == op || this.post.op == op)
			this.post = null;
	}

	if (this.blackhole && HOLED_UPDATES.indexOf(kind) >= 0)
		return;
	this.socket.write(msg);
};

const HOLED_UPDATES = [common.DELETE_POSTS, common.DELETE_THREAD];

OK.on_thread_sink = function (thread, err) {
	/* TODO */
	winston.error(thread + ' sank: ' + err);
};

const WORMHOLES = [common.SYNCHRONIZE, common.FINISH_POST];

OK.on_message = function (data) {
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
	if (this.blackhole && WORMHOLES.indexOf(type) < 0)
		return;
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
	winston.error('Error by ' + JSON.stringify(this.ident) + ': '
			+ (error || msg));
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

exports.scan_client_caps = function () {
	for (var ip in STATE.clientsByIP) {
		var ident = caps.lookup_ident(ip);
		STATE.clientsByIP[ip].forEach(function (okyaku) {
			if (!okyaku.id || !okyaku.board)
				return;
			if (ident.timeout) {
				okyaku.blackhole = true;
				return;
			}
			if (!caps.can_access_board(ident, okyaku.board)) {
				try {
					okyaku.socket.close();
				}
				catch (e) { /* bleh */ }
			}
		});
	}
};

// Push message to all clients
exports.push = function(msg){
	async.each(_.values(STATE.clients), function(client){
		try {
			client.send(msg);
		}
		catch(e){/* Client died, but we don't care */}
	});
};
