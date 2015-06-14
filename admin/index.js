var authcommon = require('./common'),
    caps = require('../server/caps'),
    common= require('../common/index'),
    okyaku = require('../server/okyaku'),
    STATE = require('../server/state');

require('./panel');

function connect() {
	return global.redis;
}

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

okyaku.dispatcher[authcommon.BAN] = function (msg, client) {
	if (!caps.can_moderate(client.ident))
		return false;
	var ip = msg[0];
	var type = msg[1];
	var sentence = msg[2];
	if (!authcommon.is_valid_ip(ip))
		return false;
	var key = authcommon.ip_key(ip);

	var m = connect().multi();
	if (!ban(m, client, ip, key, type, sentence))
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

var lift_expired_bans;
(function lift_expired_bans(){
	var r = global.redis;
	var again = setTimeout(lift_expired_bans, 60000);
	// Get banned IP hashes
	r.smembers('hot:timeouts', function(err, banned){
		if (err || !banned)
			return again;
		if (banned.length == 0)
			return again;
		var m = r.multi();
		for (i = 0; i < banned.length; i++){
			m.hgetall('ip:' + banned[i]);
		}
		m.exec(function(err, res){
			// Read and check, if ban has expired
			var m = r.multi();
			var must_reload;
			var now = Date.now();
			var ADDRS = authcommon.modCache.addresses;
			for (i = 0; i < banned.length; i++){
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
			if (must_reload){
				m.publish('reloadHot', 'caps');
				m.exec();
			}
			return again;
		});
	});
})();
