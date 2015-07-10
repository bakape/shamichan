/*
Core  server-side administration module
 */

'use strict';

let authcommon = require('./common'),
    caps = require('../server/caps'),
    common = require('../common'),
	config = require('../config'),
    okyaku = require('../server/okyaku'),
	mnemonics = require('./mnemonic/mnemonics'),
	Muggle = require('../util/etc').Muggle;

require('./panel');

function connect() {
	return global.redis;
}

let mnemonizer = new mnemonics.mnemonizer(config.SECURE_SALT);

function genMnemonic(ip) {
	return ip && mnemonizer.Apply_mnemonic(ip);
}
exports.genMnemonic = genMnemonic;

function ban(m, mod, ip, key, type, sentence) {
	if (type == 'unban') {
		// unban from every type of suspension
		authcommon.suspensionKeys.forEach(function (suffix) {
			m.srem('hot:' + suffix, key);
		});
		m.hdel('ip:' + key, 'ban', 'sentence');
	}
	else {
		// need to validate that this is a valid ban type
		// TODO: elaborate
		if (type != 'timeout')
			return false;
			
		var till = (sentence == 'perma') ? sentence : Date.now() + sentence;
		m.sadd('hot:' + type + 's', key);
		m.hmset('ip:' + key, 'ban', type, 'sentence', till);
	}
	var now = Date.now();
	var info = {ip: key, type: type, time: now, 'sentence': till};
	if (key !== ip)
		info.realip = ip;
	if (mod.ident.email)
		info.email = mod.ident.email;
	m.rpush('auditLog', JSON.stringify(info));

	// trigger reload
	m.publish('reloadHot', 'caps');

	return true;
}

let dispatcher = okyaku.dispatcher;

dispatcher[authcommon.BAN] = function (msg, client) {
	if (!caps.can_moderate(client.ident))
		return false;
	const ip = msg[0],
		type = msg[1],
		sentence = msg[2];
	if (!authcommon.is_valid_ip(ip))
		return false;
	const key = authcommon.ip_key(ip);

	let m = connect().multi();
	if (!ban(m, client, ip, key, type, sentence))
		return false;

	m.exec(function (err) {
		if (err)
			return client.kotowaru(err);
		const wasBanned = type !== 'unban';

		/* XXX not DRY */
		let ADDRS = authcommon.modCache.addresses;
		if (ADDRS[key])
			ADDRS[key].ban = wasBanned;

		client.send([0, common.MODEL_SET, ['addrs', key], {ban: wasBanned}]);
	});
	return true;
};

function lift_expired_bans() {
	let r = global.redis;

	// Get banned IP hashes
	r.smembers('hot:timeouts', function (err, banned) {
		if (err || !banned)
			return;
		if (banned.length == 0)
			return;
		let m = r.multi();
		for (let ip of banned) {
			m.hgetall('ip:' + ip);
		}
		m.exec(function (err, res) {
			// Read and check, if ban has expired
			let m = r.multi(),
				must_reload,
				ADDRS = authcommon.modCache.addresses;
			const now = Date.now();
			for (i = 0; i < banned.length; i++) {
				if (!res[i].sentence || res[i].sentence == 'perma')
					continue;
				if (res[i].sentence < now){
					must_reload = true;
					m.srem('hot:timeouts', banned[i]);
					m.hdel('ip:' + banned[i], 'ban', 'sentence');
					if (ADDRS[banned[i]])
						ADDRS[banned[i]].ban = false;
				}
			}
			if (must_reload) {
				m.publish('reloadHot', 'caps');
				m.exec();
			}
		});
	});
}
setInterval(lift_expired_bans, 60000);
lift_expired_bans();

dispatcher[common.SPOILER_IMAGES] = caps.modHandler(function (nums, client) {
	client.db.modHandler('spoilerImages', nums, function (err) {
		if (err)
			client.kotowaru(Muggle("Couldn't spoiler images.", err));
	});
	return true;
});
