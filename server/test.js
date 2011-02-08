var db = require('./db');

var y = new db.Yakusoku();
y.on('error', function (err) {
	console.error('ERROR', err);
	y.r.quit();
});
var msg = {time: 2012, state:[0,0]};
y.insert_post(msg, 'Hello, world!', '127.0.0.1', function (err, num) {
	if (err) throw err;
	console.log("Made thread " + num);
	var reply = {time: 2013, email: 'sage', op: num, state:[0, 0]};
	y.insert_post(reply, 'reported.', '127.0.0.2', function (err, num) {
		if (err) throw err;
		console.log("Made reply " + num);

		y.on('thread', console.log.bind(console, 'T'));
		y.on('post', console.log.bind(console, 'P'));
		y.on('end', function () {
			y.r.quit();
		});
		y.get_tag();
	});
});
