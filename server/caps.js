var authcommon = require('../authcommon'),
    check = require('./msgcheck').check,
    common = require('../common'),
    config = require('../config'),
    db = require('../db'),
    hooks = require('../hooks');

function can_access_board(ident, board) {
	if (board == 'graveyard' && can_administrate(ident))
		return true;
	if (board == config.STAFF_BOARD && !can_moderate(ident))
		return false;
	if (ident.ban)
		return false;
	if (under_curfew(ident, board))
		return false;
	return db.is_board(board);
}
exports.can_access_board = can_access_board;

exports.can_access_thread = function (ident, op) {
	var tags = db.tags_of(op);
	if (!tags)
		return false;
	for (var i = 0; i < tags.length; i++)
		if (can_access_board(ident, tags[i]))
			return true;
	return false;
};

function can_moderate(ident) {
	return (ident.auth === 'Admin' || ident.auth === 'Moderator');
}
exports.can_moderate = can_moderate;

function can_administrate(ident) {
	return ident.auth === 'Admin';
}
exports.can_administrate = can_administrate;

function denote_priv(info) {
	if (info.data.priv)
		info.header.push(' (priv)');
}

function dead_media_paths(paths) {
	paths.src = '../dead/src/';
	paths.thumb = '../dead/thumb/';
}

exports.augment_oneesama = function (oneeSama, opts) {
	var ident = opts.ident;
	oneeSama.ident = ident;
	if (can_moderate(ident))
		oneeSama.hook('headerName', authcommon.ip_mnemonic);
	if (can_administrate(ident))
		oneeSama.hook('headerName', denote_priv);
	if (can_administrate(ident) && opts.board == 'graveyard')
		oneeSama.hook('mediaPaths', dead_media_paths);
};

exports.mod_handler = function (func) {
	return function (nums, client) {
		if (!can_moderate(client.ident))
			return false;
		var opts = nums.shift();
		if (!check({when: 'string'}, opts) || !check('id...', nums))
			return false;
		if (!(opts.when in authcommon.delayDurations))
			return false;
		var delay = authcommon.delayDurations[opts.when];
		if (!delay)
			func(nums, client);
		else
			setTimeout(func.bind(null, nums, client), delay*1000);
		return true;
	};
};

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
		if (bits > 0 && bits <= 32)
			info.mask = 0x100000000 - Math.pow(2, 32 - bits);
	}
	return info;
}

var hotBoxes = [];
var hotBans = [];

function parse_ranges(ranges) {
	if (!ranges)
		return [];
	ranges = ranges.map(parse_ip);
	ranges.sort(function (a, b) { return a.num - b.num; });
	return ranges;
}

function range_lookup(ranges, num) {
	/* Ideally would have a tree lookup here or something */
	var full = null;
	for (var i = 0; i < ranges.length; i++) {
		var box = ranges[i];
		/* sint32 issue here doesn't matter for realistic ranges */
		if ((box.mask ? (num & box.mask) : num) === box.num)
			full = box.full; /* fall through */
	}
	return full;
}

hooks.hook('reloadHot', function (hot, cb) {
	hotBoxes = parse_ranges(hot.BOXES);
	hotBans = parse_ranges(hot.BANS);
	cb(null);
});

exports.lookup_ident = function (ip) {
	var ident = {ip: ip};
	var num = parse_ip(ip).num;
	var ban = range_lookup(hotBans, num);
	if (ban) {
		ident.ban = ban;
		return ident;
	}
	var priv = range_lookup(hotBoxes, num);
	if (priv)
		ident.priv = priv;
	return ident;
};

function under_curfew(ident, board) {
	if (can_administrate(ident))
		return false;
	var curfew = config.CURFEW_HOURS;
	if (!curfew || (config.CURFEW_BOARDS || []).indexOf(board) < 0)
		return false;
	var hour = new Date().getUTCHours();
	return curfew.indexOf(hour) < 0;
}
exports.under_curfew = under_curfew;

exports.can_ever_access_board = function (ident, board) {
	return can_access_board(ident, board) || under_curfew(ident, board);
};

exports.curfew_ending_time = function (board) {
	var curfew = config.CURFEW_HOURS;
	if (!curfew || (config.CURFEW_BOARDS || []).indexOf(board) < 0)
		return null;
	var now = new Date();
	var tomorrow = day_after(now);
	var makeToday = hour_date_maker(now);
	var makeTomorrow = hour_date_maker(tomorrow);
	/* Dumb brute-force algorithm */
	var candidates = [];
	config.CURFEW_HOURS.forEach(function (hour) {
		candidates.push(makeToday(hour), makeTomorrow(hour));
	});
	candidates.sort(compare_dates);
	for (var i = 0; i < candidates.length; i++)
		if (candidates[i] > now)
			return candidates[i];
	return null;
};

exports.curfew_starting_time = function (board) {
	var curfew = config.CURFEW_HOURS;
	if (!curfew || (config.CURFEW_BOARDS || []).indexOf(board) < 0)
		return null;
	var now = new Date();
	var tomorrow = day_after(now);
	var makeToday = hour_date_maker(now);
	var makeTomorrow = hour_date_maker(tomorrow);
	/* Even dumber brute-force algorithm */
	var candidates = [];
	config.CURFEW_HOURS.forEach(function (hour) {
		hour = (hour + 1) % 24;
		if (config.CURFEW_HOURS.indexOf(hour) < 0)
			candidates.push(makeToday(hour), makeTomorrow(hour));
	});
	candidates.sort(compare_dates);
	for (var i = 0; i < candidates.length; i++)
		if (candidates[i] > now)
			return candidates[i];
	return null;
};

function compare_dates(a, b) {
	return a.getTime() - b.getTime();
}

function day_after(today) {
	/* Leap shenanigans? This is probably broken somehow. Yay dates. */
	var tomorrow = new Date(today.getTime() + 24*3600*1000);
	if (tomorrow.getUTCDate() == today.getUTCDate())
		tomorrow = new Date(tomorrow.getTime() + 12*3600*1000);
	return tomorrow;
}

function hour_date_maker(date) {
	var prefix = date.getUTCFullYear() + '/' + (date.getUTCMonth()+1)
			+ '/' + date.getUTCDate() + ' ';
	return (function (hour) {
		return new Date(prefix + hour + ':00:00 GMT');
	});
}
