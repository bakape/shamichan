var async = require('async'),
    config = require('../config'),
    child_process = require('child_process'),
    db = require('./db'),
    etc = require('../util/etc'),
    fs = require('fs'),
    hooks = require('../util/hooks'),
    path = require('path'),
    winston = require('winston');

exports.Onegai = db.Onegai;
exports.config = config;

var image_attrs = ('src thumb ext dims size MD5 SHA1 hash imgnm spoiler vint'
		+ ' apng mid audio length').split(' ');
exports.image_attrs = image_attrs;

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
		var h = {
			'Cache-Control': 'no-cache, no-store',
			'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
		};
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
		image.dims = image.dims.split(',').map(parse_number);
	image.size = parse_number(image.size);
	delete image.hash;
	post.image = image;
});

function parse_number(n) {
	return parseInt(n, 10);
}

hooks.hook_sync('inlinePost', function (info) {
	var post = info.dest, image = info.src.image;
	if (!image)
		return;
	image_attrs.forEach(function (key) {
		if (key in image)
			post[key] = image[key];
	});
});

function publish(alloc, cb) {
	var mvs = [];
	for (var kind in alloc.tmps) {
		var src = media_path('tmp', alloc.tmps[kind]);

		var destDir = kind;
		var destKey = kind;
		var dest = media_path(destDir, alloc.image[destKey]);

		mvs.push(etc.cpx.bind(etc, src, dest));
	}
	async.parallel(mvs, cb);
}

function validate_alloc(alloc) {
	if (!alloc || !alloc.image || !alloc.tmps)
		return;
	for (var dir in alloc.tmps) {
		var fnm = alloc.tmps[dir];
		if (!/^[\w_]+$/.test(fnm)) {
			winston.warn("Suspicious filename: "
					+ JSON.stringify(fnm));
			return;
		}
	}
	return true;
}


hooks.hook("buryImage", function (info, callback) {
	if (!info.src)
		return callback(null);
	/* Just in case */
	var m = /^\d+\w*\.\w+$/;
	if (!info.src.match(m))
		return callback(etc.Muggle('Invalid image.'));
	var mvs = [mv.bind(null, 'src', info.src)];
	function try_thumb(path, t) {
		if (!t)
			return;
		if (!t.match(m))
			return callback(etc.Muggle('Invalid thumbnail.'));
		mvs.push(mv.bind(null, path, t));
	}
	try_thumb('thumb', info.thumb);
	try_thumb('mid', info.mid);
	async.parallel(mvs, callback);
	function mv(p, nm, cb) {
		etc.movex(media_path(p, nm), dead_path(p, nm), cb);
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
	var dir = base ? path.join(base, key) : config.MEDIA_DIRS[key];
	etc.checked_mkdir(dir, cb);
}
exports._make_media_dir = make_dir;

exports.make_media_dirs = function (cb) {
	var keys = ['src', 'thumb', 'vint', 'dead'];
	if (!is_standalone())
		keys.push('tmp');
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
};

exports.serve_image = function (req, resp) {
	var m = /^\/(src|thumb|mid|vint)(\/\d+\.\w+)$/.exec(req.url);
	if (!m)
		return false;
	var root = config.MEDIA_DIRS[m[1]];
	if (!root)
		return false;
	require('send')(req, m[2], {root: root}).pipe(resp);
	return true;
};

exports.squish_MD5 = function (hash) {
	if (typeof hash == 'string')
		hash = new Buffer(hash, 'hex');
	return hash.toString('base64').replace(/\//g, '_').replace(/=*$/, '');
};

exports.obtain_image_alloc = function (id, cb) {
	var onegai = new db.Onegai;
	onegai.obtain_image_alloc(id, function (err, alloc) {
		onegai.disconnect();
		if (err)
			return cb(err);

		if (validate_alloc(alloc))
			cb(null, alloc);
		else
			cb("Invalid image alloc");
	});
};

exports.commit_image_alloc = function (alloc, cb) {
	publish(alloc, function (err) {
		if (err)
			return cb(err);

		var o = new db.Onegai;
		o.commit_image_alloc(alloc, function (err) {
			o.disconnect();
			cb(err);
		});
	});
};

var is_standalone = exports.is_standalone = db.is_standalone;
