const async = require('async'),
	common = require('../common'),
	compare = require('bindings')('compare').hashCompareCpp,
	config = require('../config'),
	db = require('../db'),
	events = require('events'),
	fs = require('fs'),
	Muggle = require('../util/etc').Muggle,
	tail = require('../util/tail'),
	winston = require('winston');

const IMG_EXPIRY = 60;
let redis = global.redis;

class Onegai extends events.EventEmitter {
	constructor() {
		super();
	}
	track_temporary(path, cb) {
		redis.sadd('temps', path, (err, tracked) => {
			if (err)
				return cb(err);
			if (tracked > 0)
				setTimeout(() => this.del_temp(path), (IMG_EXPIRY + 1) * 1000);
			cb(null);
		});
	}
	lose_temporaries(files, cb) {
		redis.srem('temps', files, cb);
	}
	del_temp(path) {
		this.cleanup_image_alloc(path, function(err) {
			if (err)
				winston.warn(`unlink ${path}: ${err}`);
		});
	}
	// if an image doesn't get used in a post in a timely fashion, delete it
	cleanup_image_alloc(path, cb) {
		redis.srem('temps', path, function(err, n) {
			if (err)
				return winston.warn(err);
			if (n) {
				fs.unlink(path, function(err) {
					if (err)
						return cb(err);
					cb(null, true);
				});
			}
			else
				cb(null, false); // wasn't found
		});
	}
	// catch any dangling images on server startup
	delete_temporaries(callback) {
		redis.smembers('temps', function(err, temps) {
			if (err)
				return callback(err);
			async.each(temps,
				function (temp, cb) {
					fs.unlink(temp, function(err) {
						if (err)
							winston.warn('temp: ' + err);
						else
							winston.info('del temp ' + temp);
						cb();
					});
				},
				function() {
					redis.del('temps', callback);
				}
			);
		});
	}
	check_duplicate(image, callback) {
		redis.zrangebyscore('imageDups', Date.now(), '+inf',
			function(err, hashes) {
				if (err)
					return callback(err);
				if (!hashes)
					return callback(false);

				// Compare image hashes with C++ addon
				let isDup = compare(config.DUPLICATE_THRESHOLD, image, hashes);
				if (isDup) {
					isDup = Muggle(common.parseHTML
						`Duplicate of 
						<a href="./${isDup}" class="history" target="_blank">
							>>${isDup}
						</a>`
					);
				}
				callback(isDup);
			}
		);
	}
	record_image_alloc(id, alloc, callback) {
		redis.setex('image:' + id, IMG_EXPIRY, JSON.stringify(alloc), callback);
	}
	obtain_image_alloc(id, callback) {
		let m = redis.multi();
		const key = 'image:' + id;
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
			let alloc = JSON.parse(rs[0]);
			alloc.id = id;
			callback(null, alloc);
		});
	}
	commit_image_alloc(alloc, cb) {
		// We should already hold the lock at this point.
		const key = 'image:' + alloc.id;
		let m = redis.multi();
		m.del(key);
		m.del('lock:' + key);
		m.exec(cb);
	}
	client_message(client_id, msg) {
		redis.publish('client:' + client_id, JSON.stringify(msg));
	}
	relay_client_messages() {
		let redis = db.redis_client();
		redis.psubscribe('client:*');
		redis.once('psubscribe', () => {
			this.emit('relaying');
			redis.on('pmessage', (pat, chan, message) => {
				const id = parseInt(chan.match(/^client:(\d+)$/)[1], 10);
				this.emit('message', id, JSON.parse(message));
			});
		});
	}
}
exports.Onegai = Onegai;

// Remove expired duplicate image hashes
function cleanUpDups() {
	redis.zremrangebyscore('imageDups', 0, Date.now(), function (err) {
		if (err)
			winston.error('Error cleaning up expired image duplicates:', err);
	});
}
setInterval(cleanUpDups, 60000);
