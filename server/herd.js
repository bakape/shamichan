var config = require('./config'),
    db = require('./db');

var R;
function connect() {
	if (!R) {
		R = db.redis_client();
		R.on('error', function (err) {
			console.error(err);
			process.exit(1);
		});
	}
	return R;
}

function at_next_minute(func) {
	var now = new Date().getTime();
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

function clean_up() {
	var r = connect();
	var expiryKey = db.expiry_queue_key();
	var now = Math.floor(new Date().getTime() / 1000);
	r.zrangebyscore(expiryKey, 1, now, 'limit', 0, 10,
				function (err, expired) {
		if (err) {
			console.error(err);
			return;
		}
		expired.forEach(function (entry) {
			var m = entry.match(/^(\d+):/);
			if (!m)
				return;
			var op = parseInt(m[1], 10);
			if (!op)
				return;
			console.log('CULL', op);
		});
	});
}

if (require.main === module) {
	// TEMP
	clean_up();

	connect();
	at_next_minute(clean_up);
}
