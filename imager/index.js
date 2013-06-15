var async = require('async'),
    config = require('./config'),
    child_process = require('child_process'),
    db = require('./db'),
    fs = require('fs'),
    hooks = require('../hooks'),
    Muggle = require('../muggle').Muggle,
    path = require('path'),
    winston = require('winston');

exports.Onegai = db.Onegai;
exports.config = config;

var image_attrs = ('src thumb dims size MD5 hash imgnm spoiler realthumb vint'
		+ ' apng mid').split(' ');
exports.image_attrs = image_attrs;

function mv_file(src, dest, callback) {
	child_process.execFile('/bin/mv', ['-n', src, dest],
				function (err, stdout, stderr) {
		if (err)
			callback(Muggle("Couldn't move file into place.",
					stderr || err));
		else
			callback(null);
	});
}
exports.mv_file = mv_file;

exports.send_dead_image = function (kind, filename, resp) {
	filename = dead_path(kind, filename);
	var stream = fs.createReadStream(filename);
	stream.once('error', function (err) {
		if (err.code == 'ENOENT') {
			resp.writeHead(404);
			resp.end('Image not found');
		}
		else {
			winston.error(err);
			resp.end();
		}
	});
	stream.once('open', function () {
		var h = {};
		try {
			h['Content-Type'] = require('mime').lookup(filename);
		} catch (e) {}
		resp.writeHead(200, h);
		stream.pipe(resp);
	});
};

hooks.hook_sync('extractPost', function (post) {
	if (!is_image(post))
		return;
	var image = {};
	image_attrs.forEach(function (key) {
		if (key in post) {
			image[key] = post[key];
			delete post[key];
		}
	});
	if (image.dims.split)
		image.dims = image.dims.split(',');
	image.size = parseInt(image.size);
	delete image.hash;
	post.image = image;
});

hooks.hook_sync('inlinePost', function (info) {
	var post = info.dest, image = info.src.image;
	if (!image)
		return;
	image_attrs.forEach(function (key) {
		if (key in image)
			post[key] = image[key];
	});
});

hooks.hook("buryImage", function (info, callback) {
	if (!info.src)
		return callback(null);
	/* Just in case */
	var m = /^\d+\w*\.\w+$/;
	if (!info.src.match(m))
		return callback(Muggle('Invalid image.'));
	var mvs = [mv.bind(null, 'src', info.src)];
	function try_thumb(path, t) {
		if (!t)
			return;
		if (!t.match(m))
			return callback(Muggle('Invalid thumbnail.'));
		mvs.push(mv.bind(null, path, t));
	}
	try_thumb('thumb', info.thumb);
	try_thumb('thumb', info.realthumb);
	try_thumb('mid', info.mid);
	async.parallel(mvs, callback);
	function mv(p, nm, cb) {
		mv_file(media_path(p, nm), dead_path(p, nm), cb);
	}
});

function is_image(image) {
	return image && (image.src || image.vint);
}

function media_path(dir, filename) {
	return path.join(config.MEDIA_DIRS[dir], filename);
}
exports.media_path = media_path;

function dead_path(dir, filename) {
	return path.join(config.MEDIA_DIRS.dead, dir, filename);
}

function make_dir(base, key, cb) {
	var dir;
	if (base)
		dir = path.join(base, key);
	else
		dir = config.MEDIA_DIRS[key];
	fs.stat(dir, function (err, info) {
		var make = false;
		if (err) {
			if (err.code == 'ENOENT')
				make = true;
			else
				return cb(err);
		}
		else if (!info.isDirectory())
			return cb(dir + " is not a directory");
		if (make)
			fs.mkdir(dir, cb);
		else
			cb(null);
	});
}

exports.make_media_dirs = function (cb) {
	var keys = ['src', 'thumb', 'vint', 'dead', 'tmp'];
	if (config.EXTRA_MID_THUMBNAILS)
		keys.push('mid');
	async.forEach(keys, make_dir.bind(null, null), function (err) {
		if (err)
			return cb(err);
		var dead = config.MEDIA_DIRS.dead;
		var keys = ['src', 'thumb'];
		if (config.EXTRA_MID_THUMBNAILS)
			keys.push('mid');
		async.forEach(keys, make_dir.bind(null, dead), cb);
	});
}

exports.squish_MD5 = function (hash) {
	if (typeof hash == 'string')
		hash = new Buffer(hash, 'hex');
	return hash.toString('base64').replace(/\//g, '_').replace(/=*$/, '');
};

/* Dumb forwards */
exports.obtain_image_alloc = function (id, cb) {
	var onegai = new db.Onegai;
	onegai.obtain_image_alloc(id, function (err, alloc) {
		onegai.disconnect();
		cb(err, alloc);
	});
};

exports.make_image_nontemporary = db.make_image_nontemporary;
