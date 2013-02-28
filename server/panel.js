var _ = require('../lib/underscore'),
    caps = require('./caps'),
    common = require('../common'),
    okyaku = require('./okyaku'),
    STATE = require('./state');

okyaku.dispatcher[common.SUBSCRIBE] = function (msg, client) {
	if (!caps.can_administrate(client.ident))
		return false;
	if (msg[0] != 'adminState')
		return false;

	var byIP = {};
	for (var ip in STATE.clientsByIP)
		byIP[ip] = STATE.clientsByIP[ip].length;

	var state = {
		ips: byIP,
		uptime: process.uptime(),
		memoryUsage: process.memoryUsage(),
		visible: true,
	};
	client.send([0, common.MODEL_SET, 'adminState', state]);
	return true;
};

okyaku.dispatcher[common.UNSUBSCRIBE] = function (msg, client) {
	if (!caps.can_administrate(client.ident))
		return false;
	if (msg[0] != 'adminState')
		return false;
	client.send([0, common.MODULE_SET, 'adminState', {visible: false}]);
};
