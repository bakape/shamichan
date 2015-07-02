'use strict';

// Define vars both on server and client
var _, common, config, DEF,
	isNode = typeof navigator === 'undefined';

if (isNode) {
	_ = require('underscore');
	common = require('../common/index');
	config = require('../config');
	DEF = exports;
}
else {
	_ = window._;
	common = window;
	config = window.config;
	DEF = window.DEF;
}

DEF.FETCH_ADDRESS = 101;
DEF.SET_ADDRESS_NAME = 102;
DEF.BAN = 103;

var modCache = {}; // TEMP

var suspensionKeys = ['boxes', 'bans', 'slows', 'suspensions', 'timeouts'];

var delayNames = ['now', 'soon', 'later'];
var delayDurations = {now: 0, soon: 60, later: 20*60};

function denote_hidden(info) {
	if (info.data.hide)
		info.header.push(common.safe(
				' <em class="mod hidden">(hidden)</em>'));
}

function is_IPv4_ip(ip) {
	if (typeof ip != 'string' || !/^\d+\.\d+\.\d+\.\d+$/.exec(ip))
		return false;
	var nums = ip.split('.');
	for (var i = 0; i < 4; i++) {
		var n = parseInt(nums[i], 10);
		if (n > 255)
			return false;
		if (n && nums[i][0] == '0')
			return false;
	}
	return true;
}

var is_valid_ip = function (ip) {
	return typeof ip == 'string' && /^[\da-fA-F.:]{3,45}$/.test(ip);
};

function explode_IPv6_ip(ip) {
	if (typeof ip != 'string')
		return null;

	var groups = ip.split(':');
	var gap = groups.indexOf('');
	if (gap >= 0 || groups.length != 8) {
		// expand ::
		if (gap < 0 || gap != groups.lastIndexOf(''))
			return null;
		var zeroes = [gap, 1];
		for (let i = groups.length; i < 9; i++)
			zeroes.push('0');
		groups.splice.apply(groups, zeroes);
	}

	// check hex components
	for (let i = 0; i < groups.length; i++) {
		var n = parseInt(groups[i], 16);
		if (_.isNaN(n) || n > 0xffff)
			return null;
		groups[i] = n.toString(16);
	}

	return groups;
}

function ip_key(ip) {
	if (!is_IPv4_ip(ip)) {
		// chop off the last half of IPv6 ips
		var bits = explode_IPv6_ip(ip);
		if (bits && bits.length == 8)
			return bits.slice(0, 4).join(':');
	}
	return ip;
}

if (typeof IDENT != 'undefined') {
	/* client */
	oneeSama.hook('headerName', denote_hidden);
}

if (isNode){
	exports.modCache = modCache;
	exports.suspensionKeys = suspensionKeys;
	exports.delayDurations = delayDurations;
	exports.denote_hidden = denote_hidden;
	exports.is_IPv4_ip = is_IPv4_ip;
	exports.is_valid_ip = is_valid_ip;
	exports.ip_key = ip_key;
}
