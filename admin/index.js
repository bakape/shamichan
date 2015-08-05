/*
Core  server-side administration module
 */

'use strict';

let caps = require('../server/caps'),
	check = require('../server/msgcheck'),
    common = require('../common'),
	config = require('../config'),
	db = require('../db'),
	events = require('events'),
    okyaku = require('../server/okyaku'),
	mnemonics = require('./mnemonic/mnemonics'),
	Muggle = require('../util/etc').Muggle,
	winston = require('winston');

let mnemonizer = new mnemonics.mnemonizer(config.SECURE_SALT);

function genMnemonic(ip) {
	return ip && mnemonizer.Apply_mnemonic(ip);
}
exports.genMnemonic = genMnemonic;

let dispatcher = okyaku.dispatcher,
	redis = global.redis;

function modHandler(method, errMsg) {
	return function (nums, client) {
		return caps.checkAuth('janitor', client.ident)
			&& check('id...', nums)
			&& client.db.modHandler(method, nums, function (err) {
				if (err)
					client.kotowaru(Muggle(errMsg, err));
			});
	};
}

dispatcher[common.SPOILER_IMAGES] = modHandler('spoilerImages',
	'Couldn\'t spoiler images.'
);

dispatcher[common.DELETE_IMAGES] = modHandler('deleteImages',
	'Couldn\'t delete images.'
);

// Non-persistent global live admin notifications
dispatcher[common.NOTIFICATION] = function (msg, client) {
	msg = msg[0];
	if (!caps.checkAuth('admin', client.ident) || !check('string', msg))
		return false;
	okyaku.push([0, common.NOTIFICATION, common.escape_html(msg)]);
	return true;
};

dispatcher[common.MOD_LOG] = function (msg, client) {
	if (!caps.checkAuth('janitor', client.ident))
		return false;

	redis.zrange('modLog', 0, -1, function (err, log) {
		if (err)
			return winston.error('Moderation log fetch error:', err);
		if (!log.length)
			return;
		client.send([0, common.MOD_LOG, db.destrigifyList(log)]);
	});
	return true;
};
