/* Deletes the original images and thumbnails of archived posts,
 * leaving just the 'vint' thumbnail.
 */

var crypto = require('crypto'),
    db = require('../db'),
    etc = require('../util/etc'),
    fs = require('fs'),
    path = require('path'),
    imager = require('../imager'),
    tail = require('../util/tail'),
    winston = require('winston');

function Recycler() {
	this.tag = 'archive';
	this.y = new db.Yakusoku(this.tag, db.UPKEEP_IDENT);
}

var R = Recycler.prototype;

R.recycle_post = function (post, cb) {
	if (!post.image || !post.image.src || post.hideimg)
		return cb(null);
	var r = this.y.connect();
	var image = post.image;
	var src = imager.media_path('src', image.src);
	var toDelete = [];
	if (image.thumb) {
		toDelete.push(src);
		src = imager.media_path('thumb', image.thumb);
	}
	if (image.mid) {
		toDelete.push(imager.media_path('mid', image.mid));
	}

	MD5_file(src, function (err, MD5) {
		if (err) {
			if (err.code == 'ENOENT')
				winston.warn(src + " doesn't exist.");
			else
				winston.error(err);
			return cb(null);
		}
		var dest = MD5 + path.extname(src);
		var dest_path = imager.media_path('vint', dest);
		etc.movex(src, dest_path, function (err) {
			if (err)
				return cb(err);
			var m = r.multi();
			var key = post.op ? 'post:' + post.num
					: 'thread:' + post.num;
			m.hdel(key, 'src');
			m.hdel(key, 'thumb');
			m.hdel(key, 'mid');
			m.hset(key, 'vint', dest);
			m.exec(function (err) {
				if (err) {
					// move it back
					etc.movex(dest_path, src,
							function (e) {
						if (e)
							winston.error(e);
						return cb(err);
					});
					return;
				}

				toDelete.forEach(function (victim) {
					fs.unlink(victim, function (err) {
						if (err)
							winston.error(err);
					});
				});
				if (toDelete.length) {
					var info = post.num + ': del ' +
							toDelete.length;
					winston.info(info);
				}

				cb(null);
			});
		});
	});
};

R.recycle_thread = function (op, cb) {
	op = parseInt(op, 10);
	var reader = new db.Reader();
	reader.get_thread(this.tag, op, {});
	var do_post = this.recycle_post.bind(this);
	reader.on('thread', function (thread) {
		if (thread.immortal)
			return cb(null);
		// grrr, ought to stream
		var posts = [thread];
		reader.on('post', function (post) {
			posts.push(post);
		});
		reader.on('endthread', function () {
			tail.forEach(posts, do_post, cb);
		});
		reader.on('error', cb);
	});
};

R.recycle_archive = function (cb) {
	var key = 'tag:' + this.tag.length + ':' + this.tag;
	var r = this.y.connect();
	var do_thread = this.recycle_thread.bind(this);
	r.zrange(key + ':threads', 0, -1, function (err, threads) {
		if (err)
			return cb(err);
		tail.forEach(threads, do_thread, cb);
	});
};

function MD5_file(path, callback) {
	var stream = fs.createReadStream(path);
	var hash = crypto.createHash('md5');
	stream.once('error', function (err) {
		stream.destroy();
		callback(err);
	});
	stream.on('data', function (buf) {
		hash.update(buf);
	});
	stream.on('end', function () {
		stream.destroy();
		/* grr stupid digest() won't give us a Buffer */
		hash = new Buffer(hash.digest('binary'), 'binary');
		callback(null, imager.squish_MD5(hash));
	});
}

if (require.main === module) {
	var recycler = new Recycler;
	recycler.recycle_archive(function (err) {
		if (err) throw err;
		recycler.y.disconnect();
		process.exit(0);
	});
}
