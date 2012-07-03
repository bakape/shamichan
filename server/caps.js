var authcommon = require('../authcommon'),
    common = require('../common'),
    config = require('../config'),
    db = require('../db'),
    hooks = require('../hooks');

exports.can_access = function (ident, board) {
	if (board == 'graveyard' && is_admin_ident(ident))
		return true;
	if (under_curfew(ident, board))
		return false;
	return db.is_board(board);
};

function is_mod_ident(ident) {
	return (ident.auth === 'Admin' || ident.auth === 'Moderator');
}
exports.is_mod_ident = is_mod_ident;

function is_admin_ident(ident) {
	return ident.auth === 'Admin';
}
exports.is_admin_ident = is_admin_ident;

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
	if (is_mod_ident(ident))
		oneeSama.hook('headerName', authcommon.ip_mnemonic);
	if (is_admin_ident(ident))
		oneeSama.hook('headerName', denote_priv);
	if (is_admin_ident(ident) && opts.board == 'graveyard')
		oneeSama.hook('mediaPaths', dead_media_paths);
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
	var info = {ip: num};
	if (m[5]) {
		var bits = parseInt(m[5], 10);
		if (bits > 0 && bits <= 32)
			info.mask = 0x100000000 - Math.pow(2, 32 - bits);
	}
	return info;
}
exports.parse_ip = parse_ip;

exports.lookup_ident = function (ip) {
	return {ip: ip};
};

function under_curfew(ident, board) {
	if (is_admin_ident(ident))
		return false;
	var curfew = config.CURFEW_HOURS;
	if (!curfew || (config.CURFEW_BOARDS || []).indexOf(board) < 0)
		return false;
	var hour = new Date().getUTCHours();
	return curfew.indexOf(hour) < 0;
}
exports.under_curfew = under_curfew;

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
