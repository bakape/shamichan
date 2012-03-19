var common = require('../common'),
    hooks = require('../hooks');

// This is looking rather boilerplatey

hooks.hook('clientSynced', function (info, cb) {
	if (!info.live && info.count == 1) {
		var op = info.op, client = info.client;
		client.db.get_fun(op, function (err, js) {
			if (err)
				return cb(err);
			if (js)
				client.send([op, common.EXECUTE_JS, js]);
			cb(null);
		});
	}
	else
		cb(null);
});

hooks.hook('clientSynced', function (info, cb) {
	if (!info.live && info.count == 1) {
		var op = info.op, client = info.client;
		client.db.get_banner(op, function (err, msg) {
			if (err)
				return cb(err);
			if (msg)
				client.send([op, common.UPDATE_BANNER, msg]);
			cb(null);
		});
	}
	else
		cb(null);
});
