var async = require('async'),
    common = require('../common'),
    config = require('./config'),
    child_process = require('child_process'),
    imagerDb = require('./db'),
    index = require('./'),
    formidable = require('formidable'),
    fs = require('fs'),
    Muggle = require('../muggle').Muggle,
    path = require('path'),
    urlParse = require('url').parse,
    winston = require('winston');

function new_upload(req, resp) {
	var upload = new ImageUpload;
	upload.handle_request(req, resp);
}
exports.new_upload = new_upload;

function get_thumb_specs(w, h, pinky) {
	var QUALITY = config[pinky ? 'PINKY_QUALITY' : 'THUMB_QUALITY'];
	var bound = config[pinky ? 'PINKY_DIMENSIONS' : 'THUMB_DIMENSIONS'];
	var r = Math.max(w / bound[0], h / bound[1], 1);
	var bg = pinky ? '#d6daf0' : '#eef2ff';
	return {dims: [Math.round(w/r), Math.round(h/r)], quality: QUALITY,
			bg_color: bg, bound: bound};
}

var ImageUpload = function (client_id) {
	this.db = new imagerDb.Onegai;
	this.client_id = client_id;
};

var IU = ImageUpload.prototype;

var validFields = ['spoiler', 'op'];

IU.status = function (msg) {
	this.client_call('upload_status', msg);
};

IU.client_call = function (func, msg) {
	this.db.client_message(this.client_id, {func: func, arg: msg});
};

IU.handle_request = function (req, resp) {
	if (req.method.toLowerCase() != 'post') {
		resp.writeHead(400);
		resp.end("Need an upload.");
		return;
	}
	var query = req.query || urlParse(req.url, true).query;
	this.client_id = parseInt(query.id, 10);
	if (!this.client_id || this.client_id < 1) {
		resp.writeHead(400);
		resp.end("Bad client ID.");
		return;
	}

	this.resp = resp;
	var len = parseInt(req.headers['content-length'], 10);
	if (len > 0 && len > config.IMAGE_FILESIZE_MAX + (20*1024))
		return this.failure(Muggle('File is too large.'));

	var form = new formidable.IncomingForm();
	form.uploadDir = config.MEDIA_DIRS.tmp;
	form.maxFieldsSize = 50 * 1024;
	form.hash = 'md5';
	form.onPart = function (part) {
		if (part.filename && part.name == 'image')
			form.handlePart(part);
		else if (!part.filename && validFields.indexOf(part.name) >= 0)
			form.handlePart(part);
	};
	var self = this;
	form.once('error', function (err) {
		self.failure(Muggle('Upload request problem.', err));
	});
	form.once('aborted', function (err) {
		self.failure(Muggle('Upload was aborted.', err));
	});
	this.lastProgress = 0;
	form.on('progress', this.upload_progress_status.bind(this));

	try {
		form.parse(req, this.parse_form.bind(this));
	}
	catch (err) {
		self.failure(err);
	}
};

IU.upload_progress_status = function (received, total) {
	var percent = Math.floor(100 * received / total);
	var increment = (total > (512 * 1024)) ? 10 : 25;
	var quantized = Math.floor(percent / increment) * increment;
	if (quantized > this.lastProgress) {
		this.status(percent + '% received...');
		this.lastProgress = quantized;
	}
};

IU.parse_form = function (err, fields, files) {
	if (err)
		return this.failure(Muggle('Invalid upload.', err));
	if (!files.image)
		return this.failure(Muggle('No image.'));
	this.image = files.image;
	this.pinky = !!parseInt(fields.op, 10);

	var spoiler = parseInt(fields.spoiler, 10);
	if (spoiler) {
		var sps = config.SPOILER_IMAGES;
		if (sps.normal.indexOf(spoiler) < 0
				&& sps.trans.indexOf(spoiler) < 0)
			return this.failure(Muggle('Bad spoiler.'));
		this.image.spoiler = spoiler;
	}

	this.image.MD5 = squish_MD5(this.image.hash);
	this.image.hash = null;

	this.db.track_temporaries([this.image.path], null,
			this.process.bind(this));
};

IU.process = function (err) {
	if (err)
		winston.warn("Temp tracking error: " + err);
	if (this.failed)
		return;
	var image = this.image;
	image.ext = path.extname(image.filename).toLowerCase();
	if (image.ext == '.jpeg')
		image.ext = '.jpg';
	if (['.png', '.jpg', '.gif'].indexOf(image.ext) < 0)
		return this.failure(Muggle('Invalid image format.'));
	image.imgnm = image.filename.substr(0, 256);

	this.status('Verifying...');
	var tagged_path = image.ext.replace('.', '') + ':' + image.path;
	var self = this;
	var checks = {
		stat: fs.stat.bind(fs, image.path),
		dims: identify.bind(null, tagged_path),
	};
	if (image.ext == '.png')
		checks.apng = detect_APNG.bind(null, image.path);
	async.parallel(checks, verified);

	function verified(err, rs) {
		if (err)
			return self.failure(Muggle('Bad image.', err));
		var w = rs.dims.width, h = rs.dims.height;
		image.size = rs.stat.size;
		image.dims = [w, h];
		if (!w || !h)
			return self.failure(Muggle('Bad image dimensions.'));
		if (w > config.IMAGE_WIDTH_MAX && h > config.IMAGE_HEIGHT_MAX)
			return self.failure(Muggle('Image is too wide'
					+ ' and too tall.'));
		if (w > config.IMAGE_WIDTH_MAX)
			return self.failure(Muggle('Image is too wide.'));
		if (h > config.IMAGE_HEIGHT_MAX)
			return self.failure(Muggle('Image is too tall.'));
		if (rs.apng)
			image.apng = 1;

		perceptual_hash(tagged_path, image, function (err, hash) {
			if (err)
				return self.failure(err);
			image.hash = hash;
			self.db.check_duplicate(image.hash, deduped);
		});
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
		var media_path = index.media_path, mv_file = index.mv_file;
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
			if (err)
				return self.failure(Muggle("Distro failure.",
						err));
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

IU.read_image_filesize = function (callback) {
	var self = this;
	fs.stat(this.image.path, function (err, stat) {
		if (err)
			callback(Muggle('Internal filesize error.', err));
		else if (stat.size > config.IMAGE_FILESIZE_MAX)
			callback(Muggle('File is too large.'));
		else
			callback(null, stat.size);
	});
};

function which(name, callback) {
	child_process.exec('which ' + name, function (err, stdout, stderr) {
		if (err)
			throw err;
		callback(stdout.trim());
	});
}

/* Look up imagemagick paths */
var identifyBin, convertBin;
which('identify', function (bin) { identifyBin = bin; });
which('convert', function (bin) { convertBin = bin; });

function identify(taggedName, callback) {
	var m = taggedName.match(/^(\w{3,4}):/);
	var kind = m && m[1];
	child_process.execFile(identifyBin, [taggedName],
				function (err, stdout, stderr) {
		if (err) {
			var msg = "Bad image.";
			if (stderr.match(/no such file/i))
				msg = "Image went missing.";
			else if (stderr.match(/improper image header/i)) {
				kind = kind ? 'a ' + kind.toUpperCase()
						: 'an image';
				msg = 'File is not ' + kind + '.';
			}
			else if (stderr.match(/no decode delegate/i))
				msg = "Unsupported file type.";
			return callback(Muggle(msg, stderr));
		}

		var line = stdout.trim();
		/* Remove filename first to avoid confusing filenames */
		var name = taggedName;
		if (line.substr(0, name.length) == name)
			line = line.substr(name.length);
		if (line.substr(0, 2) == '=>')
			line = line.substr(2);
		if (kind) {
			name = name.substr(kind.length + 1);
			if (line.substr(0, name.length) == name)
				line = line.substr(name.length);
		}

		var m = line.match(/(\d+)x(\d+)/);
		if (!m)
			callback(Muggle("Couldn't read image dimensions."));
		else
			callback(null, {width: parseInt(m[1], 10),
					height: parseInt(m[2], 10)});
	});
}

function convert(args, callback) {
	child_process.execFile(convertBin, args, function (err,stdout,stderr) {
		callback(err ? (stderr || err) : null);
	});
}

function squish_MD5(hash) {
	if (typeof hash == 'string')
		hash = new Buffer(hash, 'hex');
	return hash.toString('base64').replace(/\//g, '_').replace(/=*$/, '');
}
exports.squish_MD5 = squish_MD5;

function perceptual_hash(src, image, callback) {
	var tmp = path.join(config.MEDIA_DIRS.tmp,
			'hash' + common.random_id() + '.gray');
	var args = [src + '[0]'];
	if (image.dims.width > 1000 || image.dims.height > 1000)
		args.push('-sample', '800x800');
	args.push('-background', 'white', '-mosaic', '+matte',
			'-scale', '16x16!',
			'-type', 'grayscale', '-depth', '8',
			tmp);
	convert(args, function (err) {
		if (err)
			return callback(Muggle('Hashing error.', err));
		var bin = path.join(__dirname, 'perceptual');
		child_process.execFile(bin, [tmp],
					function (err, stdout, stderr) {
			fs.unlink(tmp);
			if (err)
				return callback(Muggle('Hashing error.',
						stderr || err));
			var hash = stdout.trim();
			if (hash.length != 64)
				return callback(Muggle('Hashing problem.'));
			callback(null, hash);
		});
	});
}

function detect_APNG(fnm, callback) {
	var bin = path.join(__dirname, 'findapng');
	child_process.execFile(bin, [fnm], function (err, stdout, stderr) {
		if (err)
			return callback(Muggle('APNG detector problem.',
					stderr || err));
		else if (stdout.match(/^APNG/))
			return callback(null, true);
		else if (stdout.match(/^PNG/))
			return callback(null, false);
		else
			return callback(Muggle('APNG detector acting up.',
					stderr || err));
	});
}

function setup_im_args(o, args) {
	var args = ['-limit', 'memory', '32', '-limit', 'map', '64'];
	var dims = o.dims;
	var samp = dims[0]*2 + 'x' + dims[1]*2;
	if (o.ext == '.jpg')
		args.push('-define', 'jpeg:size=' + samp);
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
	args.push(o.src);
	if (o.ext != '.jpg')
		args.push('-sample', samp);
	args.push('-gamma', '0.454545', '-filter', 'box');
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
	convert(args, function (err) {
		if (err)
			callback(Muggle("Resizing error.", err));
		else
			callback(null);
	});
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

IU.failure = function (err) {
	var err_desc = 'Unknown image processing error.'
	if (err instanceof Muggle) {
		err_desc = err.most_precise_error_message();
		err = err.deepest_reason();
	}
	/* Don't bother logging PEBKAC errors */
	if (!(err instanceof Muggle))
		winston.error(err);

	if (this.resp) {
		this.resp.writeHead(500, {'Content-Type': 'text/plain'});
		this.resp.end(err_desc);
		this.resp = null;
	}
	if (!this.failed) {
		this.client_call('upload_error', err_desc);
		this.failed = true;
	}
	if (this.image) {
		var files = image_files(this.image);
		files.forEach(function (file) {
			fs.unlink(file, function (err) {
				if (err)
					winston.warn("Deleting " +
						file + ": " + err);
			});
		});
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
	index.image_attrs.forEach(function (key) {
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
		self.client_call('on_image_alloc', image_id);
		self.db.disconnect();
		if (self.resp) {
			self.resp.writeHead(202);
			self.resp.end('OK');
			self.resp = null;
		}
	});
};

if (require.main == module) {
	require('http').createServer(new_upload).listen(config.UPLOAD_PORT);
}
