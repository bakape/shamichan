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
	var client = info.client;
	client.db.get_banner(function (err, banner) {
		if (err)
			return cb(err);
		if (!banner || banner.tag != client.board)
			return cb(null);
		var msg = banner.message;
		if (msg)
			client.send([banner.op, common.UPDATE_BANNER, msg]);
		cb(null);
	});
});
