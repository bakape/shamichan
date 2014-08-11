var authcommon = require('./common'),
    caps = require('../server/caps'),
    common= require('../common'),
    okyaku = require('../server/okyaku'),
    STATE = require('../server/state');

require('./panel');

function connect() {
	return global.redis;
}

function ban(m, mod, ip, key, type) {
	if (type == 'unban') {
		// unban from every type of suspension
		authcommon.suspensionKeys.forEach(function (suffix) {
			m.srem('hot:' + suffix, key);
		});
		m.hdel('ip:' + key, 'ban');
	}
	else {
		// need to validate that this is a valid ban type
		// TODO: elaborate
		if (type != 'timeout')
			return false;

		m.sadd('hot:' + type + 's', key);
		m.hset('ip:' + key, 'ban', type);
	}
	var now = Date.now();
	var info = {ip: key, type: type, time: now};
	if (key !== ip)
		info.realip = ip;
	if (mod.ident.email)
		info.email = mod.ident.email;
	m.rpush('auditLog', JSON.stringify(info));

	// trigger reload
	m.publish('reloadHot', 'caps');

	return true;
}

okyaku.dispatcher[authcommon.BAN] = function (msg, client) {
	if (!caps.can_administrate(client.ident))
		return false;
	var ip = msg[0];
	var type = msg[1];
	if (!authcommon.is_valid_ip(ip))
		return false;
	var key = authcommon.ip_key(ip);

	var m = connect().multi();
	if (!ban(m, client, ip, key, type))
		return false;

	m.exec(function (err) {
		if (err)
			return client.kotowaru(err);
		var wasBanned = type != 'unban';

		/* XXX not DRY */
		var ADDRS = authcommon.modCache.addresses;
		if (ADDRS[key])
			ADDRS[key].ban = wasBanned;

		var a = {ban: wasBanned};
		client.send([0, common.MODEL_SET, ['addrs', key], a]);
	});
	return true;
};
