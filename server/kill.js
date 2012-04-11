var config = require('../config');

var cfg = config.DAEMON;
if (!cfg)
	throw "No daemon config.";

var lock = require('path').join(cfg.PID_PATH, 'server.pid');
require('daemon').kill(lock, function (err) {
	if (err)
		throw err;
});
