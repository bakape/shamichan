const compare = require('bindings')('compare').hashCompareCpp,
	config = require('../config'),
	common = require('../common'),
	events = require('events'),
	fs = require('fs'),
	Muggle = require('../util/etc').Muggle,
	winston = require('winston')

const IMG_EXPIRY = 60,
	{redis} = global

export class ClientController extends events.EventEmitter {
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
					isDup = Muggle(
						'Duplicate of '
						+`<a href="./${isDup}" class="history" target="_blank">`
							+`>>${isDup}`
						+`</a>`
					);
				}
				callback(isDup);
			}
		);
	}
	record_image_alloc(id, alloc, callback) {
		redis.setex('image:' + id, IMG_EXPIRY, JSON.stringify(alloc), callback);
	}
	client_message(client_id, msg) {
		redis.publish('client:' + client_id, JSON.stringify(msg));
	}
}

// Remove expired duplicate image hashes
function cleanUpDups() {
	redis.zremrangebyscore('imageDups', 0, Date.now(), function (err) {
		if (err)
			winston.error('Error cleaning up expired image duplicates:', err);
	});
}
setInterval(cleanUpDups, 60000);
