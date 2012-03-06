
var HOOKS = {};

exports.hook = function (key, func) {
	var hs = HOOKS[key];
	if (!hs)
		hs = HOOKS[key] = [];
	hs.push(func);
};

exports.trigger = function (key, arg) {
	var hs = HOOKS[key];
	if (hs)
		hs.forEach(function (hook) {
			hook(arg);
		});
};
