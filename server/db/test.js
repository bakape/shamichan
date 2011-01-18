var db = require('./index');

var msg = {time: 2012, name: 'guy'};
db.insert_post(msg, 'Hello, world!', '127.0.0.1', function (err, num) {
	if (err) throw err;
	console.log("Made thread " + num);
	var reply = {time: 2013, email: 'sage', op: num};
	db.insert_post(reply, 'reported.', '127.0.0.2', function (err, num) {
		if (err) throw err;
		console.log("Made reply " + num);

		reader = new db.Reader();
		reader.on('post', console.log.bind(console));
		reader.on('end', console.log.bind(console, 'Done.'));
		reader.get_tag('moe');
	});
});
