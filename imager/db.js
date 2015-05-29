'use strict';

var common = require('../common'),
	compare = require('./compare.node').hashCompareCpp,
	config = require('../config'),
	db = require('../db'),
	events = require('events'),
	fs = require('fs'),
	Muggle = require('../util/etc').Muggle,
	tail = require('../util/tail'),
	util = require('util'),
	winston = require('winston');

const IMG_EXPIRY = 60;

function redis_client() {
	return db.redis_client();
}

function connect() {
	return global.redis;
}

function Onegai() {
	events.EventEmitter.call(this);
}

util.inherits(Onegai, events.EventEmitter);
exports.Onegai = Onegai;
var O = Onegai.prototype;

O.disconnect = function() {};

O.track_temporary = function(path, cb) {
	var m = connect();
	var self = this;
	m.sadd('temps', path, function(err, tracked) {
		if (err)
			return cb(err);
		if (tracked > 0) {
			setTimeout(self.del_temp.bind(self, path),
				(IMG_EXPIRY + 1) * 1000);
		}
		cb(null);
	});
};

O.lose_temporaries = function(files, cb) {
	connect().srem('temps', files, cb);
};

O.del_temp = function(path) {
	this.cleanup_image_alloc(path, function(err, deleted) {
		if (err) {
			winston.warn('unlink ' + path + ': ' + err);
		}
	});
};

// if an image doesn't get used in a post in a timely fashion, delete it
O.cleanup_image_alloc = function(path, cb) {
	var r = connect();
	r.srem('temps', path, function(err, n) {
		if (err)
			return winston.warn(err);
		if (n) {
			fs.unlink(path, function(err) {
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
O.delete_temporaries = function(callback) {
	var r = connect();
	r.smembers('temps', function(err, temps) {
		if (err)
			return callback(err);
		tail.forEach(temps, function(temp, cb) {
			fs.unlink(temp, function(err) {
				if (err)
					winston.warn('temp: ' + err);
				else
					winston.info('del temp ' + temp);
				cb(null);
			});
		}, function(err) {
			if (err)
				return callback(err);
			r.del('temps', callback);
		});
	});
};

O.check_duplicate = function(image, callback) {
	connect().zrangebyscore('imageDups',
		Date.now(),
		'+inf',
		function(err, hashes) {
			if (err)
				return callback(err);
			if (!hashes)
				return callback(false);
			// Compare image hashes with C++ addon
			var isDup = compare(config.DUPLICATE_THRESHOLD, image, hashes);
			if (isDup)
				isDup = Muggle(common.parseHTML
					`Duplicate of
					<a href="./${isDup}" class="history" target="_blank">
						>>${isDup}
					</a>`
				);
			callback(isDup);
		}
	);
};

// Remove expired duplicate image hashes
function cleanUpDups() {
	connect().zremrangebyscore('imageDups', 0, Date.now());
}
setInterval(cleanUpDups, 60000);

O.record_image_alloc = function(id, alloc, callback) {
	var r = connect();
	r.setex('image:' + id, IMG_EXPIRY, JSON.stringify(alloc), callback);
};

O.obtain_image_alloc = function(id, callback) {
	var m = connect().multi();
	var key = 'image:' + id;
	m.get(key);
	m.setnx('lock:' + key, '1');
	m.expire('lock:' + key, IMG_EXPIRY);
	m.exec(function(err, rs) {
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

O.commit_image_alloc = function(alloc, cb) {
	// We should already hold the lock at this point.
	var key = 'image:' + alloc.id;
	var m = connect().multi();
	m.del(key);
	m.del('lock:' + key);
	m.exec(cb);
};

O.client_message = function(client_id, msg) {
	connect().publish('client:' + client_id, JSON.stringify(msg));
};

O.relay_client_messages = function() {
	var r = redis_client();
	r.psubscribe('client:*');
	var self = this;
	r.once('psubscribe', function() {
		self.emit('relaying');
		r.on('pmessage', function(pat, chan, message) {
			var id = parseInt(chan.match(/^client:(\d+)$/)[1], 10);
			self.emit('message', id, JSON.parse(message));
		});
	});
};
