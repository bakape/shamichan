var async = require('async'),
    common = require('../common'),
    config = require('../config'),
    child_process = require('child_process'),
    db = require('../db'),
    formidable = require('formidable'),
    fs = require('fs'),
    hooks = require('../hooks'),
    im = require('imagemagick'),
    path = require('path'),
    util = require('util'),
    winston = require('winston');

var image_attrs = ('src thumb dims size MD5 hash imgnm spoiler realthumb vint'
		+ ' apng').split(' ');

function is_image(image) {
	return image && (image.src || image.vint);
};

hooks.hook('extractPost', function (post, cb) {
	if (!is_image(post))
		return cb(null);
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
	cb(null);
});

hooks.hook('inlinePost', function (info, cb) {
	var post = info.dest, image = info.src.image;
	if (!image)
		return cb(null);
	image_attrs.forEach(function (key) {
		if (key in image)
			post[key] = image[key];
	});
	cb(null);
});

function get_thumb_specs(w, h, pinky) {
	var QUALITY = config[pinky ? 'PINKY_QUALITY' : 'THUMB_QUALITY'];
	var bound = config[pinky ? 'PINKY_DIMENSIONS' : 'THUMB_DIMENSIONS'];
	var r = Math.max(w / bound[0], h / bound[1], 1);
	var bg = pinky ? '#d6daf0' : '#eef2ff';
	return {dims: [Math.round(w/r), Math.round(h/r)], quality: QUALITY,
			bg_color: bg, bound: bound};
}

exports.ImageUpload = function (db, status) {
	this.db = db;
	this.statusCallback = status;
};

var IU = exports.ImageUpload.prototype;

var validFields = ['client_id', 'spoiler', 'op'];

IU.status = function (msg) {
	if (this.client_id)
		this.statusCallback.call(null, this.client_id, msg);
};

IU.handle_request = function (req, resp, board) {
	this.board = board;
	this.resp = resp;
	var accepts = (req.headers.accept || '').split(',');
	for (var i = 0; i < accepts.length; i++) {
		var mime = accepts[i].split(';')[0].trim();
		if (mime == 'application/json') {
			this.json_response = true;
			break;
		}
	}
	var len = parseInt(req.headers['content-length'], 10);
	if (len > 0 && len > config.IMAGE_FILESIZE_MAX + (20*1024))
		return this.failure('File is too large.');
	var form = new formidable.IncomingForm();
	form.maxFieldsSize = 2048;
	form.onPart = function (part) {
		if (part.filename && part.name == 'image')
			form.handlePart(part);
		else if (!part.filename && validFields.indexOf(part.name) >= 0)
			form.handlePart(part);
		else
			this._error('Superfluous field.');
	};
	try {
		form.parse(req, this.parse_form.bind(this));
	}
	catch (err) {
		winston.error(err);
		this.failure("Invalid request.");
	}
};

IU.parse_form = function (err, fields, files) {
	if (err) {
		winston.error("Upload error: " + err);
		return this.failure('Invalid upload.');
	}
	if (!files.image)
		return this.failure('No image.');
	this.image = files.image;
	this.client_id = fields.client_id;
	this.pinky = !!fields.op;

	var spoiler = parseInt(fields.spoiler, 10);
	if (spoiler) {
		var sps = config.SPOILER_IMAGES;
		if (sps.normal.indexOf(spoiler) < 0
				&& sps.trans.indexOf(spoiler) < 0)
			return this.failure('Bad spoiler.');
		this.image.spoiler = spoiler;
	}

	this.db.track_temporaries([this.image.path], null,
			this.process.bind(this));
};

IU.process = function (err) {
	if (err)
		winston.warn("Temp tracking error: " + err);
	var image = this.image;
	image.ext = path.extname(image.filename).toLowerCase();
	if (image.ext == '.jpeg')
		image.ext = '.jpg';
	if (['.png', '.jpg', '.gif'].indexOf(image.ext) < 0)
		return this.failure('Invalid image format.');
	image.imgnm = image.filename.substr(0, 256);

	this.status('Verifying...');
	var tagged_path = image.ext.replace('.', '') + ':' + image.path;
	var self = this;
	var checks = {
		stat: fs.stat.bind(fs, image.path),
		dims: im.identify.bind(im, tagged_path),
	};
	if (image.ext == '.png')
		checks.apng = detect_APNG.bind(null, image.path);
	async.parallel(checks, verified);

	function verified(err, rs) {
		if (err) {
			winston.error(err);
			return self.failure('Bad image.');
		}
		var w = rs.dims.width, h = rs.dims.height;
		image.size = rs.stat.size;
		image.dims = [w, h];
		if (!w || !h)
			return self.failure('Invalid image dimensions.');
		if (w > config.IMAGE_WIDTH_MAX)
			return self.failure('Image is too wide.');
		if (h > config.IMAGE_HEIGHT_MAX)
			return self.failure('Image is too tall.');
		if (rs.apng)
			image.apng = 1;

		async.parallel({
			MD5: MD5_file.bind(null, image.path),
			hash: perceptual_hash.bind(null, tagged_path)
		}, hashed);
	}

	function hashed(err, rs) {
		if (err)
			return self.failure(err);
		image.MD5 = rs.MD5;
		image.hash = rs.hash;
		self.db.check_duplicate(image.hash, deduped);
	}

	function deduped(err, rs) {
		if (err)
			return self.failure(err);
		image.thumb_path = image.path + '_thumb';
		var pinky = self.pinky;
		var w = image.dims[0], h = image.dims[1];
		var specs = get_thumb_specs(w, h, pinky);
		/* Determine if we really need a thumbnail */
		var sp = image.spoiler;
		if (!sp && image.size < 30*1024
				&& ['.jpg', '.png'].indexOf(image.ext) >= 0
				&& !image.apng
				&& w <= specs.dims[0] && h <= specs.dims[1]) {
			return got_thumbnail(false, false, null);
		}
		var info = {
			src: tagged_path,
			ext: image.ext,
			dest: image.thumb_path,
			dims: specs.dims,
			quality: specs.quality,
			bg: specs.bg_color,
		};
		if (sp && config.SPOILER_IMAGES.trans.indexOf(sp) >= 0) {
			self.status('Spoilering...');
			var comp = composite_src(sp, pinky);
			image.comp_path = image.path + '_comp';
			image.dims = [w, h].concat(specs.bound);
			info.composite = comp;
			info.compDest = image.comp_path;
			info.compDims = specs.bound;
			async.parallel([resize_image.bind(null, info, false),
				resize_image.bind(null, info, true)],
				got_thumbnail.bind(null, true, comp));
		}
		else {
			image.dims = [w, h].concat(specs.dims);
			if (!sp)
				self.status('Thumbnailing...');
			resize_image(info, false,
					got_thumbnail.bind(null, true, false));
		}
	}

	function got_thumbnail(nail, comp, err) {
		if (err)
			return self.failure(err);
		self.status('Publishing...');
		var time = new Date().getTime();
		image.src = time + image.ext;
		var dest, mvs;
		dest = media_path('src', image.src);
		mvs = [mv_file.bind(null, image.path, dest)];
		if (nail) {
			nail = time + '.jpg';
			image.thumb = nail;
			nail = media_path('thumb', nail);
			mvs.push(mv_file.bind(null, image.thumb_path, nail));
		}
		if (comp) {
			comp = time + 's' + image.spoiler + '.jpg';
			image.composite = comp;
			comp = media_path('thumb', comp);
			mvs.push(mv_file.bind(null, image.comp_path, comp));
			delete image.spoiler;
		}
		async.parallel(mvs, function (err, rs) {
			if (err) {
				winston.error(err);
				return self.failure("Distro failure.");
			}
			var olds = [image.path];
			var news = [dest];
			image.path = dest;
			if (nail) {
				image.thumb_path = nail;
				news.push(nail);
			}
			if (comp) {
				image.comp_path = comp;
				news.push(comp);
			}
			self.db.track_temporaries(news, olds,
					self.record_image.bind(self));
		});
	}
}

function composite_src(spoiler, pinky) {
	var file = 'spoiler' + (pinky ? 's' : '') + spoiler + '.png';
	return path.join('www', 'kana', file);
}

function media_path(dir, filename) {
	return path.join(config.MEDIA_DIRS[dir], filename);
}
exports.media_path = media_path;

IU.read_image_filesize = function (callback) {
	var self = this;
	fs.stat(this.image.path, function (err, stat) {
		if (err) {
			winston.error(err);
			callback('Internal filesize error.');
		}
		else if (stat.size > config.IMAGE_FILESIZE_MAX)
			callback('File is too large.');
		else
			callback(null, stat.size);
	});
};

function MD5_file(path, callback) {
	child_process.exec('md5sum -b ' + path, function (err, stdout, stderr) {
		if (err)
			return callback('Hashing error.');
		var m = stdout.match(/^([\da-f]{32})/i);
		if (!m)
			return callback('Hashing error.');
		var hash = new Buffer(m[1], 'hex').toString('base64');
		if (!hash)
			return callback('Hashing error.');
		hash = hash.replace(/\//g, '_');
		callback(null, hash.replace(/=*$/, ''));
	});
}
exports.MD5_file = MD5_file;

function mv_file(src, dest, callback) {
	var mv = child_process.spawn('/bin/mv', ['-n', src, dest]);
	mv.on('error', callback);
	mv.stderr.on('data', function (buf) {
		process.stderr.write(buf);
	});
	mv.on('exit', function (code) {
		callback(code ? 'mv error' : null);
	});
}
exports.mv_file = mv_file;

function perceptual_hash(src, callback) {
	var tmp = '/tmp/hash' + common.random_id() + '.gray';
	var args = [src + '[0]',
			'-background', 'white', '-mosaic', '+matte',
			'-scale', '16x16!',
			'-type', 'grayscale', '-depth', '8',
			tmp];
	im.convert(args, function (err, stdout, stderr) {
		if (err) {
			winston.error(stderr);
			return callback('Hashing error.');
		}
		var bin = path.join(__dirname, 'perceptual');
		var hasher = child_process.spawn(bin, [tmp]);
		hasher.on('error', function (err) {
			fs.unlink(tmp);
			callback(err);
		});
		var hash = [];
		hasher.stdout.on('data', function (buf) {
			hash.push(buf.toString('ascii'));
		});
		hasher.stderr.on('data', function (buf) {
			process.stderr.write(buf);
		});
		hasher.on('exit', function (code) {
			fs.unlink(tmp);
			if (code != 0)
				return callback('Hashing error.');
			hash = hash.join('').trim();
			if (hash.length != 64)
				return callback('Hashing problem.');
			callback(null, hash);
		});
	});
}

function detect_APNG(fnm, callback) {
	var bin = path.join(__dirname, 'findapng');
	child_process.exec(bin + ' ' + fnm, function (err, stdout, stderr) {
		if (err)
			return callback(stderr);
		else if (stdout.match(/^APNG/))
			return callback(null, true);
		else if (stdout.match(/^PNG/))
			return callback(null, false);
		else
			return callback(stderr);
	});
}

hooks.hook("buryImage", function (info, callback) {
	if (!info.src)
		return callback(null);
	/* Just in case */
	var m = /^\d+\w*\.\w+$/;
	if (!info.src.match(m))
		return callback('Invalid image.');
	var mvs = [mv.bind(null, 'src', info.src)];
	function try_thumb(t) {
		if (!t)
			return;
		if (!t.match(m))
			return callback('Invalid thumbnail.');
		mvs.push(mv.bind(null, 'thumb', t));
	}
	try_thumb(info.thumb);
	try_thumb(info.realthumb);
	async.parallel(mvs, callback);
	function mv(p, nm, cb) {
		mv_file(media_path(p, nm),
			path.join(config.MEDIA_DIRS.dead, p, nm), cb);
	}
});

function setup_im_args(o, args) {
	var args = [], dims = o.dims;
	if (o.ext == '.jpg')
		args.push('-define', 'jpeg:size=' + (dims[0] * 2) + 'x' +
				(dims[1] * 2));
	if (!o.setup) {
		o.src += '[0]';
		o.dest = 'jpg:' + o.dest;
		if (o.compDest)
			o.compDest = 'jpg:' + o.compDest;
		o.flatDims = o.dims[0] + 'x' + o.dims[1];
		if (o.compDims)
			o.compDims = o.compDims[0] + 'x' + o.compDims[1];
		o.quality += '';
		o.setup = true;
	}
	args.push(o.src, '-gamma', '0.454545', '-filter', 'box');
	return args;
}

function resize_image(o, comp, callback) {
	var args = setup_im_args(o);
	var dims = comp ? o.compDims : o.flatDims;
	args.push('-resize', dims + (comp ? '^' : '!'));
	args.push('-gamma', '2.2', '-background', o.bg);
	if (comp)
		args.push(o.composite, '-layers', 'flatten', '-extent', dims);
	else
		args.push('-layers', 'mosaic', '+matte');
	args.push('-strip', '-quality', o.quality, comp ? o.compDest : o.dest);
	im.convert(args, im_callback.bind(null, callback));
}

function im_callback(cb, err, stdout, stderr) {
	if (err) {
		winston.error(stderr);
		return cb('Conversion error.');
	}
	if (config.DEBUG)
		setTimeout(cb, 1000);
	else
		cb();
}

function image_files(image) {
	var files = [];
	if (image.path)
		files.push(image.path);
	if (image.thumb_path)
		files.push(image.thumb_path);
	if (image.comp_path)
		files.push(image.comp_path);
	return files;
}

IU.failure = function (err_desc) {
	this.form_call('upload_error', err_desc);
	if (this.image) {
		var files = image_files(this.image);
		files.forEach(fs.unlink.bind(fs));
		this.db.track_temporaries(null, files, function (err) {
			if (err)
				winston.warn("Tracking failure: " + err);
		});
	}
	this.db.disconnect();
};

IU.record_image = function (err) {
	if (err)
		winston.warn("Tracking failure: " + err);
	var view = {};
	var self = this;
	image_attrs.forEach(function (key) {
		if (key in self.image)
			view[key] = self.image[key];
	});
	if (this.image.composite) {
		view.realthumb = view.thumb;
		view.thumb = this.image.composite;
	}
	view.pinky = this.pinky;
	var image_id = common.random_id().toFixed();
	var alloc = {image: view, paths: image_files(this.image)};
	this.db.record_image_alloc(image_id, alloc, function (err) {
		if (err)
			return this.failure("Publishing failure.");
		self.form_call('on_image_alloc', image_id);
		self.db.disconnect();
	});
};

IU.form_call = function (func, param) {
	var resp = this.resp;
	if (this.json_response) {
		resp.writeHead(200, {'Content-Type': 'application/json'});
		resp.end(JSON.stringify({func: func, arg: param}));
		return;
	}
	param = param ? JSON.stringify(param) : '';
	resp.writeHead(200, {'Content-Type': 'text/html; charset=UTF-8'});
	resp.end('<!doctype html>\n<title></title>\n<script>'
		+ 'parent.postForm.' + func + '(' + param + ');</script>');
};
