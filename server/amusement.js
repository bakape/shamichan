var common = require('../common'),
    hooks = require('../hooks');

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
