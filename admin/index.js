/*
Core  server-side administration module
 */

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

function modHandler(kind, auth, errMsg) {
	return function (nums, client) {
		return caps.checkAuth(auth, client.ident)
			&& check('id...', nums)
			&& client.db.modHandler(kind, nums, function (err) {
				if (err)
					client.kotowaru(Muggle(errMsg, err));
			});
	};
}

dispatcher[common.SPOILER_IMAGES] = modHandler(common.SPOILER_IMAGES, 'janitor',
	'Couldn\'t spoiler images:');

dispatcher[common.DELETE_IMAGES] = modHandler(common.DELETE_IMAGES, 'janitor',
	'Couldn\'t delete images:');

dispatcher[common.DELETE_POSTS] = modHandler(common.DELETE_POSTS, 'janitor',
	'Couldn\'t delete posts:');

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
		client.send([0, common.MOD_LOG, db.destrigifyList(log)]);
	});
	return true;
};

// Clean up moderation log entries older than one week
function cleanLog() {
	redis.zremrangebyscore('modLog', 0, Date.now() - 1000*60*60*24*7,
		function (err) {
			if (err)
				winston.error('Error cleaning up moderation log:', err);
		}
	);
}
setInterval(cleanLog, 60000);
