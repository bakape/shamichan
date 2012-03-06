var async = require('async'),
    config = require('./config'),
    db = require('./db'),
    fs = require('fs'),
    path = require('path'),
    pix = require('./pix');

var r = db.redis_client();

r.multi().keys('post:*').keys('thread:*').exec(function (err, keys) {
	if (err) throw err;
	keys = keys[0].concat(keys[1]);
	async.forEachSeries(keys, function (key, cb) {
		if (!key.match(/^(?:post|thread):\d+$/))
			return cb(null);
		console.log(key);
		r.hget(key, 'MD5', function (err, MD5) {
			if (err)
				return cb(err);
			if (!MD5)
				return cb(null);
			console.log('< ' + MD5);
			if (MD5.length != 32)
				return cb(null);
			MD5 = new Buffer(MD5, 'hex').toString('base64');
			MD5 = MD5.replace(/=*$/, '');
			console.log('> ' + MD5);
			r.hset(key, 'MD5', MD5, cb);
		});
	}, function (err) {
		console.log('done.');
		r.quit();
	});
});
