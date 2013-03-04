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

function on_refresh(info) {
	this.send([0, common.MODEL_SET, 'adminState', info]);
}

var panelListeners = 0, panelInterval = 0;

function listen_panel(client) {
	STATE.emitter.on('change:clientsByIP', client.on_client_ip);
	STATE.emitter.on('refresh', client.on_refresh);

	panelListeners++;
	if (panelListeners == 1) {
		panelInterval = setInterval(refresh_panel_state, 10*1000);
	}
}

function unlisten_panel(client) {
	STATE.emitter.removeListener('change:clientsByIP',client.on_client_ip);
	STATE.emitter.removeListener('refresh', client.on_refresh);

	panelListeners--;
	if (panelListeners == 0) {
		clearInterval(panelInterval);
		panelInterval = 0;
	}
}

function refresh_panel_state() {
	STATE.emitter.emit('refresh', {
		memoryUsage: process.memoryUsage(),
		uptime: process.uptime(),
	});
}

function subscribe() {
	if (this.on_client_ip)
		return false;

	this.on_client_ip = on_client_ip.bind(this);
	this.on_refresh = on_refresh.bind(this);
	this.unsubscribe_admin_state = unsubscribe.bind(this);
	this.once('close', this.unsubscribe_admin_state);
	listen_panel(this);

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

	unlisten_panel(this);
	this.removeListener('close', this.unsubscribe_admin_state);
	this.on_client_ip = null;
	this.on_refresh = null;
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
