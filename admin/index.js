/*
Core  server-side administration module
 */

'use strict';

let caps = require('../server/caps'),
	check = require('../server/msgcheck'),
    common = require('../common'),
	config = require('../config'),
	db = require('../db'),
	events = require('events'),
    okyaku = require('../server/okyaku'),
	mnemonics = require('./mnemonic/mnemonics'),
	Muggle = require('../util/etc').Muggle,
	winston = require('winston');

let mnemonizer = new mnemonics.mnemonizer(config.SECURE_SALT);

function genMnemonic(ip) {
	return ip && mnemonizer.Apply_mnemonic(ip);
}
exports.genMnemonic = genMnemonic;

let dispatcher = okyaku.dispatcher,
	redis = global.redis;

function modHandler(method, errMsg) {
	return function (nums, client) {
		return caps.checkAuth('janitor', client.ident)
			&& check('id...', nums)
			&& client.db.modHandler(method, nums, function (err) {
				if (err)
					client.kotowaru(Muggle(errMsg, err));
			});
	};
}

dispatcher[common.SPOILER_IMAGES] = modHandler('spoilerImages',
	'Couldn\'t spoiler images.'
);

dispatcher[common.DELETE_IMAGES] = modHandler('deleteImages',
	'Couldn\'t delete images.'
);

// Non-persistent global live admin notifications
dispatcher[common.NOTIFICATION] = function (msg, client) {
	msg = msg[0];
	if (!caps.checkAuth('admin', client.ident) || !check('string', msg))
		return false;
	okyaku.push([0, common.NOTIFICATION, common.escape_html(msg)]);
	return true;
};

// Proxies redis publications to a set of websocket clients
class RedisDispatcher extends events.EventEmitter {
	constructor(channel, type, key, counterKey) {
		super();
		this.setMaxListeners(0);
		this.clients = new Set();
		this.key = key;
		this.type = type;

		let self = this;
		// Read message height counter from redis
		global.redis.get(counterKey, function (err, counter) {
			if (err)
				self.onError(err);
			self.counter = counter || 0;
		});

		let redis = db.redis_client();
		redis.on('error', this.onError);
		redis.on('message', this.onMessage.bind(this));
		redis.subscribe(channel);
	}
	onError(err) {
		winston.err('Mod subscription error:', err);
	}
	onMessage(chan, msg) {
		this.counter++;
		msg = JSON.parse(msg);
		for (let client of this.clients) {
			this.send(msg, client);
		}
	}
	send(msg, client) {
		client.send([0, this.type, msg])
	}
	sync(counter, client) {
		if (!caps.checkAuth('janitor', client))
			return false;

		// Fetch backlog
		const delta = this.counter - counter;
		if (delta > 0) {
			let self = this;
			redis.zrange(this.key, -delta, -1, function (err, backlog) {
				if (err)
					return self.onError(err);
				self.send(backlog, client);
			});
		}

		this.addClient(client);
		return true;
	}
	addClient(client) {
		let clients = this.clients;
		clients.add(client);
		client.once('close', function () {
			clients.delete(client);
		});
	}
}

let modLog = new RedisDispatcher('mod', common.MOD_LOG, 'modLog', 'modLogCtr');
dispatcher[common.MOD_LOG] = modLog.sync;
