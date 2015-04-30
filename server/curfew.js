var _ = require('underscore'),
    caps = require('./caps'),
    config = require('../config'),
    db = require('../db'),
    hooks = require('../util/hooks'),
    web = require('./web'),
    winston = require('winston');

var RES = require('./state').resources;

hooks.hook_sync('temporalAccessCheck', function (info) {
	if (under_curfew(info.ident, info.board))
		info.access = false;
});

hooks.hook_sync('boardDiversion', function (info) {
	if (info.diverted)
		return;
	if (under_curfew(info.ident, info.board)) {
		info.diverted = true;
		var resp = info.resp;
		resp.writeHead(200, web.noCacheHeaders);
		resp.write(RES.curfewTmpl[0]);
		resp.write('/' + info.board + '/');
		resp.write(RES.curfewTmpl[1]);
		var ending = curfew_ending_time(info.board);
		resp.write(ending ? ''+ending.getTime() : 'null');
		resp.end(RES.curfewTmpl[2]);
	}
});

function under_curfew(ident, board) {
	if (caps.can_administrate(ident))
		return false;
	var curfew = config.CURFEW_HOURS;
	if (!curfew || (config.CURFEW_BOARDS || []).indexOf(board) < 0)
		return false;
	var hour = new Date().getUTCHours();
	return curfew.indexOf(hour) < 0;
}

function curfew_ending_time(board) {
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
}

function curfew_starting_time(board) {
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
}

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

/* DAEMON */

function shutdown(board, cb) {
	var yaku = new db.Yakusoku(board, db.UPKEEP_IDENT);
	yaku.teardown(board, function (err) {
		yaku.disconnect();
		cb(err);
	});
}

function at_next_curfew_start(board, func) {
	var when = curfew_starting_time(board);
	winston.info('Next curfew for ' + board + ' at ' + when.toUTCString());
	setTimeout(func, when.getTime() - Date.now());
}

function enforce(board) {
	at_next_curfew_start(board, function () {
		winston.info('Curfew ' + board + ' at ' +
				new Date().toUTCString());
		shutdown(board, function (err) {
			if (err)
				winston.error(err);
		});
		setTimeout(enforce.bind(null, board), 30 * 1000);
	});
}

if (config.CURFEW_BOARDS && config.CURFEW_HOURS)
	config.CURFEW_BOARDS.forEach(enforce);
