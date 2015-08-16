/*
Manages client read/write permissions
 */

var async = require('async'),
    common = require('../common/index'),
    config = require('../config'),
    db = require('../db'),
    hooks = require('../util/hooks');

var RANGES = require('./state').dbCache.ranges;

function can_access_board(ident, board) {
	if (board == config.STAFF_BOARD && !common.checkAuth('janitor', ident))
		return false;
	if (ident.ban || ident.suspension)
		return false;
	return db.is_board(board);
}
exports.can_access_board = can_access_board;

function can_access_thread (ident, op) {
	var tags = db.tags_of(op);
	if (!tags)
		return false;
	for (var i = 0; i < tags.length; i++)
		if (can_access_board(ident, tags[i]))
			return tags[i];
	return false;
}
exports.can_access_thread = can_access_thread;

function dead_media_paths(paths) {
	paths.src = '../dead/src/';
	paths.thumb = '../dead/thumb/';
	paths.mid = '../dead/mid/';
}

function parse_ip(ip) {
	var m = ip.match(/^(\d+)\.(\d+)\.(\d+)\.(\d+)(?:\/(\d+))?$/);
	if (!m)
		return false;
	// damn you signed int32s!
	var num = 0;
	for (var i = 4, shift = 1; i > 0; i--) {
		num += parseInt(m[i], 10) * shift;
		shift *= 256;
	}

	var info = {full: ip, num: num};
	if (m[5]) {
		var bits = parseInt(m[5], 10);
		if (bits > 0 && bits <= 32) {
			info.mask = 0x100000000 - Math.pow(2, 32 - bits);
			info.num &= info.mask;
		}
	}
	return info;
}

function parse_ranges(ranges) {
	if (!ranges)
		return [];
	ranges = ranges.map(function (o) {
		if (typeof o == 'object') {
			o.ip = parse_ip(o.ip);
			return o;
		}
		else
			return {ip: parse_ip(o)};
	});
	ranges.sort(function (a, b) { return a.ip.num - b.ip.num; });
	return ranges;
}

function range_lookup(ranges, num) {
	if (!ranges)
		return null;
	/* Ideally would have a tree lookup here or something */
	var result = null;
	for (var i = 0; i < ranges.length; i++) {
		var box = ranges[i].ip;
		/* sint32 issue here doesn't matter for realistic ranges */
		if ((box.mask ? (num & box.mask) : num) === box.num)
			result = ranges[i];
		/* don't break out of loop */
	}
	return result;
}

function parse_suspensions(suspensions) {
	if (!suspensions)
		return [];
	var parsed = [];
	suspensions.forEach(function (s) {
		try {
			parsed.push(JSON.parse(s));
		}
		catch (e) {
			winston.error("Bad suspension JSON: " + s);
		}
	});
	return parsed;
}

function lookup_ident (ip) {
	var ident = {ip: ip};
	var num = parse_ip(ip).num;
	var ban = range_lookup(RANGES.bans, num);
	if (ban) {
		ident.ban = ban.ip.full;
		return ident;
	}
	ban = range_lookup(RANGES.timeouts, num);
	if (ban) {
		ident.ban = ban.ip.full;
		ident.timeout = true;
		return ident;
	}

	return ident;
}
exports.lookup_ident = lookup_ident;


