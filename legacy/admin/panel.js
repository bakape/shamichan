var _ = require('underscore'),
    authcommon = require('./common'),
    caps = require('../../server/caps'),
    common = require('../../common/index'),
	config = require('../../config/index'),
    okyaku = require('../../server/okyaku'),
    STATE = require('../../server/state');

var ADDRS = STATE.dbCache.addresses;
authcommon.modCache.addresses = ADDRS;
var ip_key = authcommon.ip_key;

function on_client_ip(ip, clients) {
	var addr = {key: ip_key(ip), ip: ip, count: clients.length};
	// This will leak 0-count clients.
	// I want them to expire after a delay, really. Should reduce churn.
	this.send([0, common.COLLECTION_ADD, 'addrs', addr]);
}

function on_refresh(info) {
	this.send([0, common.MODEL_SET, 'adminState', info]);
}

function connect() {
	return global.redis;
}

function address_view(addr) {
	addr = _.extend({}, addr);
	addr.shallow = false;
	var clients = STATE.clientsByIP[addr.ip];
	if (clients && clients.length)
		addr.count = clients.length;
	return addr;
}

okyaku.dispatcher[authcommon.FETCH_ADDRESS] = function (msg, client) {
	if (!caps.can_moderate(client.ident))
		return false;
	var ip = msg[0];
	if (!authcommon.is_valid_ip(ip))
		return false;
	var key = ip_key(ip);
	var addr = ADDRS[key];
	if (addr) {
		client.send([0, common.COLLECTION_ADD, 'addrs',
				address_view(addr)]);
		return true;
	}

	// Cache miss
	ADDRS[key] = addr = {ip: ip, key: key, shallow: true};
	var r = connect();
	r.hgetall('ip:'+key, function (err, info) {
		if (err) {
			if (ADDRS[key] === addr)
				delete ADDRS[key];
			return client.kotowaru(err);
		}
		if (ADDRS[key] !== addr)
			return;

		_.extend(addr, info);
		client.send([0, common.COLLECTION_ADD, 'addrs',
				address_view(addr)]);
	});
	return true;
}

okyaku.dispatcher[authcommon.SET_ADDRESS_NAME] = function (msg, client) {
	if (!caps.can_moderate(client.ident))
		return false;
	// Ignore the request, if IP tagging is disabled
	if (!config.IP_TAGGING)
		return true;
	var ip = msg[0], name = msg[1];
	if (!authcommon.is_valid_ip(ip) || typeof name != 'string')
		return false;
	var key = ip_key(ip);
	name = name.trim().slice(0, 30);
	var m = connect().multi();
	if (!name) {
		m.hdel('ip:' + key, 'name');
		m.srem('namedIPs', key);
	}
	else {
		m.hset('ip:' + key, 'name', name);
		m.sadd('namedIPs', key);
	}

	m.exec(function (err) {
		if (err)
			return client.kotowaru(err);

		// should observe a publication for this cache update
		var addr = ADDRS[key];
		if (!addr)
			addr = ADDRS[key] = {key: key, ip: ip};
		addr.name = name;

		var amend = {name: name};
		client.send([0, common.MODEL_SET, ['addrs', key], amend]);
	});
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

function snapshot_panel() {
	var addrCount = 0;
	for (var key in ADDRS)
		addrCount++;

	var ranges = STATE.dbCache.ranges;
	var banCount = ranges.bans ? ranges.bans.length : 0;

	return {
		memoryUsage: process.memoryUsage(),
		uptime: process.uptime(),
		addrs: addrCount,
		bans: banCount,
	};
}

function refresh_panel_state() {
	STATE.emitter.emit('refresh', snapshot_panel());
}

function subscribe() {
	if (this.on_client_ip)
		return false;

	this.on_client_ip = on_client_ip.bind(this);
	this.on_refresh = on_refresh.bind(this);
	this.unsubscribe_admin_state = unsubscribe.bind(this);
	this.once('close', this.unsubscribe_admin_state);
	listen_panel(this);

	var state = snapshot_panel();
	state.visible = true;
	this.send([0, common.MODEL_SET, 'adminState', state]);

	var ips = [];
	for (var ip in STATE.clientsByIP) {
		var key = ip_key(ip);
		var a = ADDRS[key];
		ips.push(a ? address_view(a) : {
			key: key, ip: ip, shallow: true,
			count: STATE.clientsByIP[ip].length
		});
	}
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
