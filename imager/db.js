var compare = require('./compare.node').hashCompareCpp,
	 config = require('../config'),
    events = require('events'),
    fs = require('fs'),
    Muggle = require('../util/etc').Muggle,
    tail = require('../util/tail'),
    util = require('util'),
    winston = require('winston');

var IMG_EXPIRY = 60;
var STANDALONE = !!config.DAEMON;

function redis_client() {
	if (STANDALONE) {
		return require('redis').createClient(config.DAEMON.REDIS_PORT);
	}
	else {
		return require('../db').redis_client();
	}
}

function Onegai() {
	events.EventEmitter.call(this);
}

util.inherits(Onegai, events.EventEmitter);
exports.Onegai = Onegai;
var O = Onegai.prototype;

O.connect = function () {
	if (STANDALONE) {
		if (!global.imagerRedis)
			global.imagerRedis = redis_client();
		return global.imagerRedis;
	}
	return global.redis;
};

O.disconnect = function () {};

O.track_temporary = function (path, cb) {
	var m = this.connect();
	var self = this;
	m.sadd('temps', path, function (err, tracked) {
		if (err)
			return cb(err);
		if (tracked > 0) {
			setTimeout(self.del_temp.bind(self, path),
				(IMG_EXPIRY+1) * 1000);
		}
		cb(null);
	});
};

O.lose_temporaries = function (files, cb) {
	this.connect().srem('temps', files, cb);
};

O.del_temp = function (path) {
	this.cleanup_image_alloc(path, function (err, deleted) {
		if (err) {
			winston.warn('unlink ' + path + ': '
					+ err);
		}
	});
};

// if an image doesn't get used in a post in a timely fashion, delete it
O.cleanup_image_alloc = function (path, cb) {
	var r = this.connect();
	r.srem('temps', path, function (err, n) {
		if (err)
			return winston.warn(err);
		if (n) {
			fs.unlink(path, function (err) {
				if (err)
					return cb(err);
				cb(null, true);
			});
		}
		else {
			cb(null, false); // wasn't found
		}
	});
};

// catch any dangling images on server startup
O.delete_temporaries = function (callback) {
	var r = this.connect();
	r.smembers('temps', function (err, temps) {
		if (err)
			return callback(err);
		tail.forEach(temps, function (temp, cb) {
			fs.unlink(temp, function (err) {
				if (err)
					winston.warn('temp: ' + err);
				else
					winston.info('del temp ' + temp);
				cb(null);
			});
		}, function (err) {
			if (err)
				return callback(err);
			r.del('temps', callback);
		});
	});
};

O.check_duplicate = function (image, callback) {
	this.connect().zrangebyscore('imageDups', Date.now(), '+inf', function (err, hashes) {
		if (err)
			return callback(err);
		if (!hashes)
			return callback(false);
		// Compare image hashes with C++ addon
		var isDup = compare(config.DUPLICATE_THRESHOLD, image, hashes);
		if (isDup)
			isDup = Muggle('Duplicate of <a href="./' + isDup + '" target="_blank">>>' + isDup + '</a>.');
		callback(isDup);
	});
};

O.record_image_alloc = function (id, alloc, callback) {
	var r = this.connect();
	r.setex('image:' + id, IMG_EXPIRY, JSON.stringify(alloc), callback);
};

O.obtain_image_alloc = function (id, callback) {
	var m = this.connect().multi();
	var key = 'image:' + id;
	m.get(key);
	m.setnx('lock:' + key, '1');
	m.expire('lock:' + key, IMG_EXPIRY);
	m.exec(function (err, rs) {
		if (err)
			return callback(err);
		if (rs[1] != 1)
			return callback(Muggle("Image in use."));
		if (!rs[0])
			return callback(Muggle("Image lost."));
		var alloc = JSON.parse(rs[0]);
		alloc.id = id;
		callback(null, alloc);
	});
};

exports.is_standalone = function () { return STANDALONE; };

O.commit_image_alloc = function (alloc, cb) {
	// We should already hold the lock at this point.
	var key = 'image:' + alloc.id;
	var m = this.connect().multi();
	m.del(key);
	m.del('lock:' + key);
	m.exec(cb);
};

O.client_message = function (client_id, msg) {
	this.connect().publish('client:' + client_id, JSON.stringify(msg));
};

O.relay_client_messages = function () {
	var r = redis_client();
	r.psubscribe('client:*');
	var self = this;
	r.once('psubscribe', function () {
		self.emit('relaying');
		r.on('pmessage', function (pat, chan, message) {
			var id = parseInt(chan.match(/^client:(\d+)$/)[1], 10);
			self.emit('message', id, JSON.parse(message));
		});
	});
};
