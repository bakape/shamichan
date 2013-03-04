var _ = require('../lib/underscore'),
    caps = require('./caps'),
    common = require('../common'),
    okyaku = require('./okyaku'),
    STATE = require('./state');

function on_client_ip(ip, clients) {
	var amend = {};
	amend[ip] = clients.length;
	this.send([0, common.MODEL_EXTEND, ['adminState', 'ips'], amend]);
}

function on_mem_usage(usage) {
	this.send([0, common.MODEL_SET, 'adminState',
			{memoryUsage: usage}]);
}

setInterval(function () {
	STATE.emitter.emit('change:memoryUsage', process.memoryUsage());
}, 10*1000);

function subscribe() {
	if (this.on_client_ip)
		return false;
	this.on_client_ip = on_client_ip.bind(this);
	this.on_mem_usage = on_mem_usage.bind(this);
	this.unsubscribe_admin_state = unsubscribe.bind(this);
	STATE.emitter.on('change:clientsByIP', this.on_client_ip);
	STATE.emitter.on('change:memoryUsage', this.on_mem_usage);
	this.once('close', this.unsubscribe_admin_state);

	var state = {
		uptime: process.uptime(),
		memoryUsage: process.memoryUsage(),
		visible: true,
	};

	state.ips = {};
	for (var ip in STATE.clientsByIP)
		state.ips[ip] = STATE.clientsByIP[ip].length;

	this.send([0, common.MODEL_SET, 'adminState', state]);
	return true;
}

function unsubscribe() {
	if (!this.on_client_ip)
		return false;
	this.removeListener('close', this.unsubscribe_admin_state);
	STATE.emitter.removeListener('change:clientsByIP', this.on_client_ip);
	STATE.emitter.removeListener('change:memoryUsage', this.on_mem_usage);
	this.on_client_ip = null;
	this.on_mem_usage = null;
	this.unsubscribe_admin_state = null;

	this.send([0, common.MODEL_SET, 'adminState', {visible: false}]);
	return true;
}

okyaku.dispatcher[common.SUBSCRIBE] = function (msg, client) {
	if (!caps.can_administrate(client.ident))
		return false;
	if (msg[0] != 'adminState')
		return false;

	return subscribe.call(client);
};

okyaku.dispatcher[common.UNSUBSCRIBE] = function (msg, client) {
	if (!caps.can_administrate(client.ident))
		return false;
	if (msg[0] != 'adminState')
		return false;
	return unsubscribe.call(client);
};
