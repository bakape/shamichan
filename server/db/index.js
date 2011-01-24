var config = require('../config'),
    flow = require('flow'),
    fs = require('fs'),
    pg = require('pg'),
    Template = require('../lib/json-template').Template,
    util = require('util');

exports.connect = pg.connect.bind(pg, config.DB_CONFIG);

exports.insert_image = function (db, image, callback) {
	var dims = image.dims;
	var values = [image.time, image.MD5, image.size,
			image.ext].concat(image.dims);
	var query = db.query({
		name: 'insert image',
		text: "INSERT INTO " + config.DB_IMAGE_TABLE +
		" (created, md5, filesize, ext, width, height, thumb_width," +
		" thumb_height, pinky_width, pinky_height) VALUES (" +
		"TIMESTAMP 'epoch' AT TIME ZONE 'UTC' + $1 * INTERVAL '1ms'," +
		" $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id",
		values: values
	});
	query.on('row', function (row) {
		callback(null, row.id);
	});
	query.on('error', function (err) {
		if (err.code == 23505) /* UNIQUE constraint */
			callback("Duplicate image detected.", null);
		else
			callback(err, null);
	});
};

exports.append_image = function (db, post_num, id, imgnm, callback) {
	var query = db.query({
		name: 'append image',
		text: "UPDATE " + config.DB_POST_TABLE +
		" SET image = $1, image_filename = $2 WHERE num = $3",
		values: [id, imgnm, post_num]
	});
	query.on('error', callback);
	query.on('end', function () { callback(null); });
};

function dims_array(f) {
	/* Stupid, stupid, stupid! */
	return [f.width, f.height, f.thumb_width, f.thumb_height,
			f.pinky_width, f.pinky_height];
};

exports.check_duplicate_image = function (db, MD5, callback) {
	var query = db.query({
		name: 'lookup image by md5',
		text: "SELECT id, filesize, ext, width, height, " +
		"thumb_width, thumb_height, pinky_width, pinky_height, " +
		"EXTRACT(epoch FROM created) * 1000 AS time FROM " +
		config.DB_IMAGE_TABLE + " WHERE md5 = $1",
		values: [MD5]
	});
	var done = false;
	query.on('row', function (r) {
		var found = {id: r.id, MD5: MD5, size: r.filesize, ext: r.ext,
				dims: dims_array(r), time: r.time};
		done = true;
		callback(null, found);
	});
	query.on('end', function () {
		if (!done)
			callback(null, null);
	});
	query.on('error', function (err) {
		callback(err, null);
	});
};

exports.update_thumbnail_dimensions = function (db, id, pinky, w, h, callback) {
	var thumb = pinky ? 'pinky' : 'thumb';
	var query = db.query({
		name: "update " + thumb + " dimensions",
		text: "UPDATE " + config.DB_IMAGE_TABLE +
		" SET "+thumb+"_width = $1, "+thumb+"_height = $2" +
		" WHERE id = $3",
		values: [w, h, id]
	});
	query.on('error', callback);
	query.on('end', function () { callback(); });
};

exports.insert_post = function(db, msg, ip, callback) {
	var query = db.query({
		name: 'insert post',
		text: "INSERT INTO " + config.DB_POST_TABLE +
		" (name, trip, email, body, parent, created, ip," +
		" image, image_filename) VALUES" +
		" ($1, $2, $3, $4, $5," +
		" TIMESTAMP 'epoch' AT TIME ZONE 'UTC' + $6 * INTERVAL '1ms',"+
		" $7, $8, $9) RETURNING num",
		values: [msg.name, msg.trip || '', msg.email || '',
			msg.body, msg.op || null, msg.time, ip,
			msg.image ? msg.image.id : null,
			msg.image ? msg.imgnm : null]
	});
	query.on('row', function (row) {
		callback(null, row.num);
	});
	query.on('error', function (err) {
		callback(err, null);
	});
}

exports.update_post = function(db, num, body, callback) {
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

exports.get_posts = function(db, get_threads, callback) {
	var postsSQL = fs.readFileSync('db/get_posts.sql', 'UTF-8');
	var vals = {DB_POST_TABLE: config.DB_POST_TABLE,
		DB_IMAGE_TABLE: config.DB_IMAGE_TABLE};
	if (!get_threads)
		vals.posts_only = true;

	var query = db.query(Template(postsSQL).expand(vals));
	var images = {};
	query.on('row', function (r) {
		var post = {num: r.num, name: r.name, trip: r.trip,
			email: r.email, body: r.body, time: r.post_time};
		if (r.parent)
			post.op = r.parent;
		var image_id = r.id;
		if (image_id) {
			var image = images[image_id];
			if (!image) {
				image = {id: image_id, MD5: r.md5,
					size: r.filesize, ext: r.ext,
					dims: dims_array(r), time: r.image_time
				};
				images[image_id] = image;
			}
			post.image = image;
			post.imgnm = r.image_filename;
		}
		callback(null, post);
	});
	query.on('error', function (error) {
		callback(error, null);
	});
	query.on('end', function () {
		callback(null, null);
	});
};

function create_table(db, table, sql_file, done) {
	console.log("Creating " + table + "...");
	var sql = fs.readFileSync(sql_file, 'UTF-8');
	var query = db.query(Template(sql).expand(config));
	query.on('end', done);
}

exports.check_tables = function (done) {
	var post = config.DB_POST_TABLE, image = config.DB_IMAGE_TABLE;
	var db = new pg.Client(config.DB_CONFIG);
	db.connect();
	flow.exec(function () {
		db.query("SELECT relname FROM pg_class " +
			"WHERE relname IN ($1, $2)", [post, image], this);
	}, function (err, exist) {
		if (err)
			throw(err);
		exist = exist.rows.map(function (row) { return row.relname; });
		if (exist.indexOf(image) < 0)
			create_table(db, image, 'db/image_table.sql', this);
		else
			this(exist);
	}, function (exist) {
		if (exist.indexOf(post) < 0)
			create_table(db, post, 'db/post_table.sql', this);
		else
			this();
	}, function () {
		db.end();
		done();
	});
};
