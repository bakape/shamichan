var config = require('./config'),
    events = require('events'),
    fs = require('fs'),
    Muggle = require('../etc').Muggle,
    tail = require('../tail'),
    util = require('util'),
    winston = require('winston');

var IMG_EXPIRY = 60;
var STANDALONE = !!config.DAEMON;

var ALLOC_CLEANUPS = {};

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

O.track_temporaries = function (adds, dels, callback) {
	var m = this.connect().multi();
	var cleans = ALLOC_CLEANUPS;
	var self = this;
	if (adds && adds.length) {
		m.sadd('temps', adds);
		adds.forEach(function (add) {
			cleans[add] = setTimeout(self.del_temp.bind(self, add),
				(IMG_EXPIRY+1) * 1000);
		});
	}
	if (dels && dels.length) {
		m.srem('temps', dels);
		dels.forEach(function (del) {
			if (del in cleans) {
				clearTimeout(cleans[del]);
				delete cleans[del];
			}
		});
	}
	m.exec(callback);
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
	delete ALLOC_CLEANUPS[path];
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

O.check_duplicate = function (hash, callback) {
	this.connect().get('hash:'+hash, function (err, num) {
		if (err)
			callback(err);
		else if (num)
			callback(Muggle('Duplicate of >>' + num + '.'));
		else
			callback(false);
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
	var cleans = ALLOC_CLEANUPS;
	alloc.paths.forEach(function (path) {
		if (path && path in cleans) {
			clearTimeout(cleans[path]);
			delete cleans[path];
			m.srem('temps', path);
		}
	});
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
