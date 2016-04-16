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
}

// Remove expired duplicate image hashes
function cleanUpDups() {
	redis.zremrangebyscore('imageDups', 0, Date.now(), function (err) {
		if (err)
			winston.error('Error cleaning up expired image duplicates:', err);
	});
}
setInterval(cleanUpDups, 60000);
