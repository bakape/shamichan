var _ = require('../lib/underscore'),
    authcommon = require('../authcommon'),
    caps = require('./caps'),
    common = require('../common'),
    okyaku = require('./okyaku'),
    STATE = require('./state');

function on_client_ip(ip, clients) {
	var addr = {ip: ip, count: clients.length};
	// This will leak 0-count clients.
	// I want them to expire after a delay, really. Should reduce churn.
	this.send([0, common.COLLECTION_ADD, 'addrs', addr]);
}

function on_refresh(info) {
	this.send([0, common.MODEL_SET, 'adminState', info]);
}

okyaku.dispatcher[authcommon.FETCH_ADDRESS] = function (msg, client) {
	if (!caps.can_moderate(client.ident))
		return false;
	var ip = msg[0];
	if (typeof ip != 'string' || !/^\d+\.\d+\.\d+\.\d+$/.exec(ip))
		return false;
	var clients = STATE.clientsByIP[ip] || [];
	var addr = {ip: ip, count: clients.length, shallow: false};
	client.send([0, common.COLLECTION_ADD, 'addrs', addr]);
	return true;
};

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

	var ips = [];
	for (var ip in STATE.clientsByIP)
		ips.push({ip: ip, count: STATE.clientsByIP[ip].length});

	this.send([0, common.MODEL_SET, 'adminState', state]);
	this.send([0, common.COLLECTION_RESET, 'addrs', ips]);
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
