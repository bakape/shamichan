// Copies old threads to the archive board
// or deletes them permenanly with config.VOLATILE

var config = require('../config'),
    db = require('../db'),
    winston = require('winston');

// Load hooks
require('../imager');
require('../server/amusement');

var yaku;
function connect() {
	var r;
	if (!yaku) {
		yaku = new db.Yakusoku('archive', db.UPKEEP_IDENT);
		r = yaku.connect();
		r.on('error', function (err) {
			winston.error(err);
		});
	}
	else
		r = yaku.connect();
	return r;
}

function at_next_minute(func) {
	var now = Date.now();
	var inFive = new Date(now + 5000);

	var nextMinute = inFive.getTime();
	var ms = inFive.getMilliseconds(), s = inFive.getSeconds();
	if (ms > 0) {
		nextMinute += 1000 - ms;
		s++;
	}
	if (s > 0 && s < 60)
		nextMinute += (60 - s) * 1000;
	var delay = nextMinute - now;

	return setTimeout(func, delay);
}

var CLEANING_LIMIT = 10; // per minute

function clean_up() {
	var r = connect();
	var expiryKey = db.expiry_queue_key();
	var now = Math.floor(Date.now() / 1000);
	r.zrangebyscore(expiryKey, 1, now, 'limit', 0, CLEANING_LIMIT,
				function (err, expired) {
		if (err) {
			winston.error(err);
			return;
		}
		expired.forEach(function (entry) {
			var m = entry.match(/^(\d+):/);
			if (!m)
				return;
			var op = parseInt(m[1], 10);
			if (!op)
				return;

			if (config.VOLATILE){
				yaku.purge_thread(op, function(){
					r.zrem(expiryKey, entry, function (err, n) {
						if (err)
							return winston.error(err);
						winston.info("Purged thread #" + op);
						if (n != 1)
							winston.warn("Not purged?");
					});
				});
			} else {
				yaku.archive_thread(op, function (err) {
					if (err)
						return winston.error(err);
					r.zrem(expiryKey, entry, function (err, n) {
						if (err)
							return winston.error(err);
						winston.info("Archived thread #" + op);
						if (n != 1)
							winston.warn("Not archived?");
					});
				});
			}
		});
	});
	at_next_minute(clean_up);
}

// Start module
connect();
at_next_minute(clean_up);
