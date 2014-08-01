var async = require('async'),
    config = require('./config'),
    child_process = require('child_process'),
    etc = require('../etc'),
    Muggle = etc.Muggle,
    imagerDb = require('./db'),
    index = require('./'),
    formidable = require('formidable'),
    fs = require('fs'),
    jobs = require('./jobs'),
    path = require('path'),
    urlParse = require('url').parse,
    util = require('util'),
    winston = require('winston');

var IMAGE_EXTS = ['.png', '.jpg', '.gif'];
if (config.WEBM) {
	IMAGE_EXTS.push('.webm');
	if (!config.DAEMON) {
		console.warn("Please enable imager.config.DAEMON security.");
	}
}

function new_upload(req, resp) {
	var upload = new ImageUpload;
	upload.handle_request(req, resp);
}
exports.new_upload = new_upload;

function get_thumb_specs(dims, pinky, scale) {
	var w = dims[0], h = dims[1];
	var quality = config[pinky ? 'PINKY_QUALITY' : 'THUMB_QUALITY'];
	var bound = config[pinky ? 'PINKY_DIMENSIONS' : 'THUMB_DIMENSIONS'];
	var r = Math.max(w / bound[0], h / bound[1], 1);
	var bg = pinky ? '#d6daf0' : '#eef2ff';
	var dims = [Math.round(w/r) * scale, Math.round(h/r) * scale];
	return {dims: dims, quality: quality, bg: bg, bound: bound};
}

var ImageUpload = function (client_id) {
	this.db = new imagerDb.Onegai;
	this.client_id = client_id;
};

var IU = ImageUpload.prototype;

var validFields = ['spoiler', 'op'];

IU.status = function (msg) {
	this.client_call('status', msg);
};

IU.client_call = function (t, msg) {
	this.db.client_message(this.client_id, {t: t, arg: msg});
};

IU.respond = function (code, msg) {
	if (!this.resp)
		return;
	this.resp.writeHead(code, {
		'Content-Type': 'text/html; charset=UTF-8',
		'Access-Control-Allow-Origin': config.MAIN_SERVER_ORIGIN,
	});
	this.resp.end('<!doctype html><title>Upload result</title>\n'
		+ 'This is a legitimate imager response.\n'
		+ '<script>\nparent.postMessage(' + JSON.stringify(msg)
		+ ', ' + JSON.stringify(config.MAIN_SERVER_ORIGIN) + ');\n'
		+ '</script>\n');
	this.resp = null;
};

IU.handle_request = function (req, resp) {
	if (req.method.toLowerCase() != 'post') {
		resp.writeHead(405, {Allow: 'POST'});
		resp.end();
		return;
	}
	this.resp = resp;
	var query = req.query || urlParse(req.url, true).query;
	this.client_id = parseInt(query.id, 10);
	if (!this.client_id || this.client_id < 1) {
		this.respond(400, "Bad client ID.");
		return;
	}

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

	this.image.MD5 = index.squish_MD5(this.image.hash);
	this.image.hash = null;

	var self = this;
	this.db.track_temporary(this.image.path, function (err) {
		if (err)
			winston.warn("Temp tracking error: " + err);
		self.process();
	});
};

IU.process = function () {
	if (this.failed)
		return;
	var image = this.image;
	var filename = image.filename || image.name;
	image.ext = path.extname(filename).toLowerCase();
	if (image.ext == '.jpeg')
		image.ext = '.jpg';
	if (IMAGE_EXTS.indexOf(image.ext) < 0)
		return this.failure(Muggle('Invalid image format.'));
	image.imgnm = filename.substr(0, 256);

	this.status('Verifying...');
	if (image.ext == '.webm')
		video_still(image.path, this.verify_webm.bind(this));
	else
		this.verify_image();
};

function video_still(src, cb) {
	var dest = index.media_path('tmp', 'still_'+etc.random_id());
	var args = ['-hide_banner', '-loglevel', 'info',
			'-i', src,
			'-f', 'image2', '-vframes', '1', '-vcodec', 'png',
			'-y', dest];
	var opts = {env: {AV_LOG_FORCE_NOCOLOR: '1'}};
	child_process.execFile(ffmpegBin, args, opts,
				function (err, stdout, stderr) {
		var lines = stderr ? stderr.split('\n') : [];
		var first = lines[0];
		if (err) {
			var msg;
			if (/no such file or directory/i.test(first))
				msg = "Video went missing.";
			else if (/invalid data found when/i.test(first))
				msg = "Invalid video file.";
			else if (/^ffmpeg version/i.test(first))
				msg = "Server's ffmpeg is too old.";
			else {
				msg = "Unknown video reading error.";
				winston.warn("Unknown ffmpeg output: "+first);
			}
			fs.unlink(dest, function (err) {
				cb(Muggle(msg, stderr));
			});
			return;
		}
		var is_webm = /matroska,webm/i.test(first);
		if (!is_webm) {
			fs.unlink(dest, function (err) {
				cb(Muggle('Video stream is not WebM.'));
			});
			return;
		}

		/* Could have false positives due to chapter titles. Bah. */
		var has_audio = /audio:\s*vorbis/i.test(stderr);

		cb(null, {
			still_path: dest,
			has_audio: has_audio,
		});
	});
}

IU.verify_webm = function (err, info) {
	if (err)
		return this.failure(err);
	var self = this;
	this.db.track_temporary(info.still_path, function (err) {
		if (err)
			winston.warn("Tracking error: " + err);

		if (info.has_audio && !config.WEBM_AUDIO)
			return self.failure(Muggle('Audio is not allowed.'));

		// pretend it's a PNG for the next steps
		var image = self.image;
		image.video = image.path;
		image.path = info.still_path;
		image.ext = '.png';
		if (info.has_audio){
			image.audio = true;
			image.spoiler = 'a';
		}
		self.verify_image();
	});
};

IU.verify_image = function () {
	var image = this.image;
	this.tagged_path = image.ext.replace('.', '') + ':' + image.path;
	var checks = {
		stat: fs.stat.bind(fs, image.video || image.path),
		dims: identify.bind(null, this.tagged_path),
	};
	if (image.ext == '.png')
		checks.apng = detect_APNG.bind(null, image.path);

	var self = this;
	async.parallel(checks, function (err, rs) {
		if (err)
			return self.failure(Muggle('Bad image.'));
		image.size = rs.stat.size;
		image.dims = [rs.dims.width, rs.dims.height];
		if (rs.apng)
			image.apng = 1;
		self.verified();
	});
};

IU.verified = function () {
	if (this.failed)
		return;
	var w = this.image.dims[0], h = this.image.dims[1];
	if (!w || !h)
		return this.failure(Muggle('Bad image dimensions.'));
	if (config.IMAGE_PIXELS_MAX && w * h > config.IMAGE_PIXELS_MAX)
		return this.failure(Muggle('Way too many pixels.'));
	if (w > config.IMAGE_WIDTH_MAX && h > config.IMAGE_HEIGHT_MAX)
		return this.failure(Muggle('Image is too wide and too tall.'));
	if (w > config.IMAGE_WIDTH_MAX)
		return this.failure(Muggle('Image is too wide.'));
	if (h > config.IMAGE_HEIGHT_MAX)
		return this.failure(Muggle('Image is too tall.'));

	var self = this;
	perceptual_hash(this.tagged_path, this.image, function (err, hash) {
		if (err)
			return self.failure(err);
		self.image.hash = hash;
		self.db.check_duplicate(hash, function (err) {
			if (err)
				return self.failure(err);
			self.deduped();
		});
	});
};

IU.fill_in_specs = function (specs, kind) {
	specs.src = this.tagged_path;
	specs.ext = this.image.ext;
	specs.dest = this.image.path + '_' + kind;
	this.image[kind + '_path'] = specs.dest;
};

IU.deduped = function (err) {
	if (this.failed)
		return;
	var image = this.image;
	var specs = get_thumb_specs(image.dims, this.pinky, 1);
	var w = image.dims[0], h = image.dims[1];

	/* Determine whether we really need a thumbnail */
	var sp = image.spoiler;
	if (!sp && image.size < 30*1024
			&& ['.jpg', '.png'].indexOf(image.ext) >= 0
			&& !image.apng && !image.video
			&& w <= specs.dims[0] && h <= specs.dims[1]) {
		return this.got_nails();
	}
	this.haveNail = true;
	this.fill_in_specs(specs, 'thumb');

	var self = this;
	if (sp && config.SPOILER_IMAGES.trans.indexOf(sp) >= 0 || image.audio) {
		this.status(image.audio ? 'Overlaying...' : 'Spoilering...');
		var comp = composite_src(sp, this.pinky);
		image.comp_path = image.path + '_comp';
		image.dims = [w, h].concat(image.audio ? specs.dims : specs.bound);
		specs.composite = comp;
		specs.compDest = image.comp_path;
		specs.compDims = specs.bound;
		specs.audio = image.audio;
		async.parallel([
			self.resize_and_track.bind(self, specs, false),
			self.resize_and_track.bind(self, specs, true),
		], function (err) {
			if (err)
				return self.failure(err);
			self.haveComp = true;
			self.got_nails();
		});
	}
	else {
		image.dims = [w, h].concat(specs.dims);
		if (!sp)
			this.status('Thumbnailing...');

		self.resize_and_track(specs, false, function (err) {
			if (err)
				return self.failure(err);

			if (config.EXTRA_MID_THUMBNAILS)
				self.middle_nail();
			else
				self.got_nails();
		});
	}
};

IU.middle_nail = function () {
	if (this.failed)
		return;

	var specs = get_thumb_specs(this.image.dims, this.pinky, 2);
	this.fill_in_specs(specs, 'mid');

	var self = this;
	this.resize_and_track(specs, false, function (err) {
		if (err)
			self.failure(err);
		self.haveMiddle = true;
		self.got_nails();
	});
};

IU.got_nails = function () {
	if (this.failed)
		return;

	var image = this.image;
	if (image.video) {
		// stop pretending this is just a still image
		image.path = image.video;
		image.ext = '.webm';
		delete image.video;
	}

	var time = Date.now();
	image.src = time + image.ext;
	var base = path.basename;
	var tmps = {src: base(image.path)};

	if (this.haveNail) {
		image.thumb = time + '.jpg';
		tmps.thumb = base(image.thumb_path);
	}
	if (this.haveMiddle) {
		image.mid = time + '.jpg';
		tmps.mid = base(image.mid_path);
	}
	if (this.haveComp) {
		image.composite = time + 's' + image.spoiler + '.jpg';
		tmps.comp = base(image.comp_path);
		delete image.spoiler;
	}

	this.record_image(tmps);
};

function composite_src(spoiler, pinky) {
	var file = 'spoiler' + (pinky ? 's' : '') + spoiler + '.png';
	return path.join(config.SPOILER_DIR, file);
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

var ffmpegBin;
if (config.WEBM) {
	which('ffmpeg', function (bin) { ffmpegBin = bin; });
}

function identify(taggedName, callback) {
	var m = taggedName.match(/^(\w{3,4}):/);
	var args = ['-format', '%Wx%H', taggedName + '[0]'];
	child_process.execFile(identifyBin, args, function (err,stdout,stderr){
		if (err) {
			var msg = "Bad image.";
			if (stderr.match(/no such file/i))
				msg = "Image went missing.";
			else if (stderr.match(/improper image header/i)) {
				var kind = m && m[1];
				kind = kind ? 'a ' + kind.toUpperCase()
						: 'an image';
				msg = 'File is not ' + kind + '.';
			}
			else if (stderr.match(/no decode delegate/i))
				msg = "Unsupported file type.";
			return callback(Muggle(msg, stderr));
		}

		var line = stdout.trim();
		var m = line.match(/(\d+)x(\d+)/);
		if (!m)
			callback(Muggle("Couldn't read image dimensions."));
		else
			callback(null, {width: parseInt(m[1], 10),
					height: parseInt(m[2], 10)});
	});
}

function ConvertJob(args, src) {
	jobs.Job.call(this);
	this.args = args;
	this.src = src;
}
util.inherits(ConvertJob, jobs.Job);

ConvertJob.prototype.perform_job = function () {
	var self = this;
	child_process.execFile(convertBin, this.args,
				function (err, stdout, stderr) {
		self.finish_job(err ? (stderr || err) : null);
	});
};

ConvertJob.prototype.describe_job = function () {
	return "ImageMagick conversion of " + this.src;
};

function convert(args, src, callback) {
	jobs.schedule(new ConvertJob(args, src), callback);
}

function perceptual_hash(src, image, callback) {
	var tmp = index.media_path('tmp',
			'hash' + etc.random_id() + '.gray');
	var args = [src + '[0]'];
	if (image.dims.width > 1000 || image.dims.height > 1000)
		args.push('-sample', '800x800');
	// do you believe in magic?
	args.push('-background', 'white', '-mosaic', '+matte',
			'-scale', '16x16!',
			'-type', 'grayscale', '-depth', '8',
			tmp);
	convert(args, src, function (err) {
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

function setup_image_params(o) {
	// only the first time!
	if (o.setup) return;
	o.setup = true;

	o.src += '[0]'; // just the first frame of the animation

	var thumbFormat = 'jpg:';
	o.dest = thumbFormat + o.dest;
	if (o.compDest)
		o.compDest = thumbFormat + o.compDest;
	o.flatDims = o.dims[0] + 'x' + o.dims[1];
	if (o.compDims)
		o.compDims = o.compDims[0] + 'x' + o.compDims[1];

	o.quality += ''; // coerce to string
}

function build_im_args(o, args) {
	// avoid OOM killer
	var args = ['-limit', 'memory', '32', '-limit', 'map', '64'];
	var dims = o.dims;
	// resample from twice the thumbnail size
	// (avoid sampling from the entirety of enormous 6000x6000 images etc)
	var samp = dims[0]*2 + 'x' + dims[1]*2;
	if (o.ext == '.jpg')
		args.push('-define', 'jpeg:size=' + samp);
	setup_image_params(o);
	args.push(o.src);
	if (o.ext != '.jpg')
		args.push('-sample', samp);
	// gamma-correct yet shitty downsampling
	args.push('-gamma', '0.454545', '-filter', 'box');
	return args;
}

function resize_image(o, comp, callback) {
	var args = build_im_args(o);
	var dims = o.audio ? o.flatDims : (comp ? o.compDims : o.flatDims);
	// in the composite case, zoom to fit. otherwise, force new size
	args.push('-resize', dims + (comp ? '^' : '!'));
	// add background
	args.push('-gamma', '2.2', '-background', o.bg);
	if (comp)
		args.push(o.composite, '-layers', 'flatten', '-extent', dims);
	else
		args.push('-layers', 'mosaic', '+matte');
	// disregard metadata, acquire artifacts
	args.push('-strip', '-quality', o.quality, comp ? o.compDest : o.dest);
	convert(args, o.src, function (err) {
		if (err) {
			winston.warn(err);
			callback(Muggle("Resizing error.", err));
		}
		else
			callback(null);
	});
}

IU.resize_and_track = function (o, comp, cb) {
	var self = this;
	resize_image(o, comp, function (err) {
		if (err)
			return cb(err);
		var fnm = comp ? o.compDest : o.dest;

		// HACK: strip IM type tag
		var m = /^\w{3,4}:(.+)$/.exec(fnm);
		if (m)
			fnm = m[1];

		self.db.track_temporary(fnm, cb);
	});
};

function image_files(image) {
	var files = [];
	if (image.path)
		files.push(image.path);
	if (image.thumb_path)
		files.push(image.thumb_path);
	if (image.mid_path)
		files.push(image.mid_path);
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

	this.respond(500, err_desc);
	if (!this.failed) {
		this.client_call('error', err_desc);
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
		this.db.lose_temporaries(files, function (err) {
			if (err)
				winston.warn("Tracking failure: " + err);
		});
	}
	this.db.disconnect();
};

IU.record_image = function (tmps) {
	if (this.failed)
		return;
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
	var image_id = etc.random_id().toFixed();
	var alloc = {image: view, tmps: tmps};
	this.db.record_image_alloc(image_id, alloc, function (err) {
		if (err)
			return this.failure("Image storage failure.");
		self.client_call('alloc', image_id);
		self.db.disconnect();
		self.respond(202, 'OK');

		if (index.is_standalone()) {
			var where = view.src;
			var size = Math.ceil(view.size / 1000) + 'kb';
			winston.info('upload: ' + where + ' ' + size);
		}
	});
};

function run_daemon() {
	var cd = config.DAEMON;
	var is_unix_socket = (typeof cd.LISTEN_PORT == 'string');
	if (is_unix_socket) {
		try { fs.unlinkSync(cd.LISTEN_PORT); } catch (e) {}
	}

	var server = require('http').createServer(new_upload);
	server.listen(cd.LISTEN_PORT);
	if (is_unix_socket) {
		fs.chmodSync(cd.LISTEN_PORT, '777'); // TEMP
	}

	index._make_media_dir(null, 'tmp', function (err) {});

	winston.info('Imager daemon listening on '
			+ (cd.LISTEN_HOST || '')
			+ (is_unix_socket ? '' : ':')
			+ (cd.LISTEN_PORT + '.'));
}

if (require.main == module) (function () {
	if (!index.is_standalone())
		throw new Error("Please enable DAEMON in imager/config.js");

	var onegai = new imagerDb.Onegai;
	onegai.delete_temporaries(function (err) {
		onegai.disconnect();
		if (err)
			throw err;
		process.nextTick(run_daemon);
	});
})();
