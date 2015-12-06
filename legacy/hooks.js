var async = require('async');
var HOOKS = {},
	SYNC_HOOKS = {};

function hook (key, func) {
	var hs = HOOKS[key];
	if (hs)
		hs.push(func);
	else
		HOOKS[key] = [func];
}
exports.hook = hook;

function trigger (key, arg, cb) {
	var hs = HOOKS[key] || [];
	async.forEachSeries(hs, function (hook, next) {
		hook(arg, next);
	}, function (err) {
		if (err)
			cb(err);
		else
			cb(null, arg);
	});
}
exports.trigger = trigger;

function hook_sync (key, func) {
	var hs = SYNC_HOOKS[key];
	if (hs)
		hs.push(func);
	else
		SYNC_HOOKS[key] = [func];
}
exports.hook_sync = hook_sync;

function trigger_sync (key, arg) {
	let hs = SYNC_HOOKS[key] || [];
	for (let i = 0, l = hs.length; i < l; i++) {
		hs[i](arg);
	}
}
exports.trigger_sync = trigger_sync;
