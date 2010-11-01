var config = require('../config').config,
    fs = require('fs'),
    jsontemplate = require('../json-template'),
    postgres = require('../../../node-postgres/lib');

var db = new postgres.Client(config.DB_CONFIG);
db.connect();
db.on('error', function (err) {
	console.log(err);
	process.exit(1);
});

exports.insert_post = function(msg, ip, callback) {
	var query = db.query({
		name: 'insert post',
		text: "INSERT INTO " + config.DB_POST_TABLE +
		" (name, trip, email, body, parent, created, ip) VALUES" +
		" ($1,$2,$3,$4,$5, TIMESTAMP 'epoch' + $6 * INTERVAL '1ms'," +
		" $7) RETURNING num",
		values: [msg.name, msg.trip || '', msg.email || '',
			msg.body, msg.op || null, msg.time, ip]
	});
	query.on('row', function (row) {
		callback(null, row.fields[0]);
	});
	query.on('error', function (err) {
		callback(err, null);
	});
}

exports.update_post = function(num, body, callback) {
	var query = db.query({
		name: 'update post',
		text: "UPDATE " + config.DB_POST_TABLE +
		" SET body = $1 WHERE num = $2",
		values: [body, num]
	});
	var done = false;
	query.on('error', function (err) {
		done = true;
		callback(false);
	});
	query.on('end', function () {
		if (!done)
			callback(true);
	});
}

exports.get_threads = function(callback) {
	var query = db.query({
		name: 'get threads',
		text: "SELECT num, name, trip, email, body, " +
		"EXTRACT(epoch FROM created) * 1000 FROM " +
		config.DB_POST_TABLE + " WHERE parent IS NULL"
	});
	query.on('row', function (row) {
		var f = row.fields;
		callback(null, {num: f[0], name: f[1], trip: f[2], email: f[3],
				body: f[4], time: f[5]});
	});
	query.on('error', function (error) {
		callback(error, null);
	});
	query.on('end', function () {
		callback(null, null);
	});
};

exports.get_posts = function(callback) {
	var query = db.query({
		name: 'get posts',
		text: "SELECT num, name, trip, email, body, parent, " +
		"EXTRACT(epoch FROM created) * 1000 FROM " +
		config.DB_POST_TABLE + " WHERE parent IS NOT NULL " +
		"ORDER BY parent, num"
	});
	query.on('row', function (row) {
		var f = row.fields;
		callback(null, {num: f[0], name: f[1], trip: f[2], email: f[3],
				body: f[4], op: f[5], time: f[6]});
	});
	query.on('error', function (error) {
		callback(error, null);
	});
	query.on('end', function () {
		callback(null, null);
	});
};

function create_table(table, sql_file, done) {
	console.log("Creating " + table + "...");
	var sql = fs.readFileSync(sql_file, 'UTF-8');
	var query = db.query(jsontemplate.Template(sql).expand(config));
	query.on('end', done);
}

exports.check_tables = function (done) {
	var post = config.DB_POST_TABLE, image = config.DB_IMAGE_TABLE;
	var exist = [];
	var query = db.query({
		text: "SELECT relname FROM pg_class WHERE relname IN ($1, $2)",
		values: [post, image]
	});
	query.on('row', function (row) {
		exist.push(row.fields[0]);
	});
	function post_table() {
		if (exist.indexOf(post) < 0)
			create_table(post, 'db/post_table.sql', done);
		else
			done();
	}
	query.on('end', function () {
		if (exist.indexOf(image) < 0)
			create_table(image, 'db/image_table.sql', post_table);
		else
			post_table();
	});
};
