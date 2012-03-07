var async = require('async');

var HOOKS = {};

exports.hook = function (key, func) {
	var hs = HOOKS[key];
	if (hs)
		hs.push(func);
	else
		HOOKS[key] = [func];
};

exports.trigger = function (key, arg, cb) {
	var hs = HOOKS[key] || [];
	async.forEachSeries(hs, function (hook, next) {
		hook(arg, next);
	}, function (err) {
		if (err)
			cb(err);
		else
			cb(null, arg);
	});
};
