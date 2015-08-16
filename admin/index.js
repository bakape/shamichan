/*
Core  server-side administration module
 */

const check = require('../server/msgcheck'),
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

function modHandler(kind, auth) {
	const errMsg = kind.replace('_', ' ').toLowerCase();
	kind = common[kind];
	dispatcher[kind] = function (nums, client) {
		return common.checkAuth(auth, client.ident)
			&& check('id...', nums)
			&& client.db.modHandler(kind, nums, function (err) {
				if (err)
					client.kotowaru(Muggle(errMsg, err));
			});
	};
}

modHandler('SPOILER_IMAGES', 'janitor');
modHandler('DELETE_IMAGES', 'janitor');
modHandler('DELETE_POSTS', 'janitor');
modHandler('LOCK_THREAD', 'moderator');
modHandler('UNLOCK_THREAD', 'moderator');

// Non-persistent global live admin notifications
dispatcher[common.NOTIFICATION] = function (msg, client) {
	msg = msg[0];
	if (!common.checkAuth('admin', client.ident) || !check('string', msg))
		return false;
	okyaku.push([0, common.NOTIFICATION, common.escape_html(msg)]);
	return true;
};

dispatcher[common.MOD_LOG] = function (msg, client) {
	if (!common.checkAuth('janitor', client.ident))
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
