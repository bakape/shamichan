var async = require('async'),
	config = require('./config'),
	configMain = require('../config'),
	crypto = require('crypto'),
	child_process = require('child_process'),
	etc = require('../etc'),
	Muggle = etc.Muggle,
	imagerDb = require('./db'),
	index = require('./'),
	findapng = require('./findapng.node'),
	formidable = require('formidable'),
	fs = require('fs'),
	jobs = require('./jobs'),
	lang = require('../lang/'),
	path = require('path'),
	urlParse = require('url').parse,
	util = require('util'),
	web = require('../server/web'),
	winston = require('winston');

var IMAGE_EXTS = ['.png', '.jpg', '.gif'];
if (config.WEBM) {
	IMAGE_EXTS.push('.webm');
	// Daemon currently broken
	/*if (!config.DAEMON) {
		console.warn("Please enable imager.config.DAEMON security.");
	}*/
}
if (config.MP3)
	IMAGE_EXTS.push('.mp3');
if (config.SVG)
	IMAGE_EXTS.push('.svg');
if (config.PDF)
	IMAGE_EXTS.push('.pdf');

function new_upload(req, resp) {
	var upload = new ImageUpload;
	upload.handle_request(req, resp);
}
exports.new_upload = new_upload;

function get_thumb_specs(image, pinky, scale) {
	const w = image.dims[0],
		h = image.dims[1],
		bound = config[pinky ? 'PINKY_DIMENSIONS' : 'THUMB_DIMENSIONS'],
		r = Math.max(w / bound[0], h / bound[1], 1),
		dims = [Math.round(w/r) * scale, Math.round(h/r) * scale];
	var specs = {bound: bound, dims: dims, format: 'jpg'};
	// Note: WebMs pretend to be PNGs at this step,
	//       but those don't need transparent backgrounds.
	//       (well... WebMs *can* have alpha channels...)
	if (config.PNG_THUMBS && image.ext == '.png' && !image.video) {
		specs.format = 'png';
		specs.quality = config.PNG_THUMB_QUALITY;
	}
	else if (pinky) {
		specs.bg = '#d6daf0';
		specs.quality = config.PINKY_QUALITY;
	}
	else {
		specs.bg = '#eef2ff';
		specs.quality = config.THUMB_QUALITY;
	}
	return specs;
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
	const query = req.query || urlParse(req.url, true).query;

	// Set response language
	// Check if client language is set and exixts on the server
	this.lang = lang[configMain.LANGS[
			web.parse_cookie(req.headers.cookie[lang])
		] || configMain.DEFAULT_LANG];

	this.client_id = parseInt(query.id, 10);
	if (!this.client_id || this.client_id < 1) {
		this.respond(400, this.lang.im_bad_client);
		return;
	}

	const len = parseInt(req.headers['content-length'], 10);
	if (len > 0 && len > config.IMAGE_FILESIZE_MAX + (20*1024))
		return this.failure(Muggle(this.lang.im_too_large));

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
		self.failure(Muggle(self.lang.im_req_problem, err));
	});
	form.once('aborted', function (err) {
		self.failure(Muggle(self.lang.im_aborted, err));
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
	const increment = (total > (512 * 1024)) ? 10 : 25,
		quantized = Math.floor(percent / increment) * increment;
	if (quantized > this.lastProgress) {
		this.status(percent + this.lang.im_received);
		this.lastProgress = quantized;
	}
};

IU.parse_form = function (err, fields, files) {
	if (err)
		return this.failure(Muggle(this.lang.im_invalid, err));
	if (!files.image)
		return this.failure(Muggle(this.lang.im_no_image));
	this.image = files.image;
	this.pinky = !!parseInt(fields.op, 10);

	const spoiler = parseInt(fields.spoiler, 10);
	if (spoiler) {
		if (config.SPOILER_IMAGES.indexOf(spoiler) < 0)
			return this.failure(Muggle(this.lang.im_bad_spoiler));
		this.image.spoiler = spoiler;
	}

	this.image.MD5 = index.squish_MD5(this.image.hash);
	this.image.hash = null;

	var self = this;
	this.db.track_temporary(this.image.path, function (err) {
		if (err)
			winston.warn(self.lang.im_temp_tracking + err);
		self.process();
	});
};

IU.process = function () {
	if (this.failed)
		return;
	var image = this.image;
	const filename = image.filename || image.name;
	image.ext = path.extname(filename).toLowerCase();
	if (image.ext == '.jpeg')
		image.ext = '.jpg';
	if (IMAGE_EXTS.indexOf(image.ext) < 0)
		return this.failure(Muggle(this.lang.im_invalid_format));
	image.imgnm = filename.substr(0, 256);

	this.status(this.lang.im_verifying);
	if (image.ext == '.webm' || image.ext == '.mp3')
		video_still(image.path, this.verify_webm.bind(this));
	else
		this.verify_image();
};

function StillJob(src) {
	jobs.Job.call(this);
	this.src = src;
}
util.inherits(StillJob, jobs.Job);

StillJob.prototype.describe_job = function () {
	return "FFmpeg video still of " + this.src;
};

StillJob.prototype.perform_job = function () {
	const dest = index.media_path('tmp', 'still_' + etc.random_id());
	var self = this;
	child_process.execFile(ffmpegBin, [
			'-hide_banner', '-loglevel', 'info',
			'-i', this.src,
			'-f', 'image2',
			'-vf', 'thumbnail=10', '-vframes', '1', '-vcodec', 'png',
			'-y', dest
		],
		{env: {AV_LOG_FORCE_NOCOLOR: '1'}},
		function (err, stdout, stderr) {
			const lines = (stderr ? stderr.split('\n') : [])[0],
				is_webm = /matroska,webm/i.test(first),
				isMP3 = /mp3/i.test(first);
			if (err) {
				var msg;
				if (/no such file or directory/i.test(first))
					msg = "Video went missing.";
				else if (/invalid data found when/i.test(first))
					msg = "Invalid video file.";
				else if (/^ffmpeg version/i.test(first))
					msg = "Server's ffmpeg is too old.";
				else if (isMP3)
					msg = 'MP3 has no cover art.';
				else {
					msg = "Unknown video reading error.";
					winston.warn("Unknown ffmpeg output: "+first);
				}
				fs.unlink(dest, function (err) {
					self.finish_job(Muggle(msg, stderr));
				});
				return;
			}
			if (!is_webm  && !isMP3) {
				fs.unlink(dest, function (err) {
					self.finish_job(Muggle('File format corrupted.'));
				});
				return;
			}

			// Parse webm/mp3 length
			var length;
			const l = stderr.match(/Duration: (\d{2}:\d{2}:\d{2})/);
			if (l){
				var h = l[1].slice(0, 3),
					m = l[1].slice(3,6);
				h = (h == '00:') ? '' : h.replace(':', 'h');
				m = (m == '00:') ? '' : m.replace(':', 'm');
				length = h + m + l[1].slice(6) + 's';
			}

			self.finish_job(null, {
				still_path: dest,
				// Could have false positives due to chapter titles. Bah.
				has_audio: (is_webm && /audio:\s*vorbis/i.test(stderr)) || isMP3,
				length: length,
				mp3: isMP3
			});
		}
	);
};

function video_still(src, cb) {
	jobs.schedule(new StillJob(src), cb);
}

IU.verify_webm = function (err, info) {
	if (err)
		return this.failure(err);
	var self = this;
	this.db.track_temporary(info.still_path, function (err) {
		if (err)
			winston.warn("Tracking error: " + err);

		if (info.has_audio && !config.WEBM_AUDIO)
			return self.failure(Muggle(self.lang.im_audio_kinshi));

		// pretend it's a PNG for the next steps
		var image = self.image;
		image.video = image.path;
		image.path = info.still_path;
		image.ext = '.png';
		if (info.has_audio)
			image.audio = true;
		if (info.length)
			image.length = info.length;
		if (info.mp3)
			image.mp3 = true;

		self.verify_image();
	});
};

IU.verify_image = function () {
	var image = this.image,
		stream = fs.createReadStream(image.path);
	this.tagged_path = image.ext.replace('.', '') + ':' + image.path;

	var self = this;
	stream.once('err', function(err) {
		winston.error(err);
		stream.close();
		self.failure(Muggle(err));
	});
	var checks = {
		// Get more accurate filesize. Formidable passes the gzipped one
		stat: fs.stat.bind(fs, image.video || image.path),
		dims: identify.bind(null, stream),
		hash: perceptual_hash.bind(null, stream)
	};
	if (image.ext == '.png')
		checks.apng = detectAPNG.bind(null, stream);

	async.parallel(checks, function (err, rs) {
		if (err)
			/*
			 * All functions, except for fs.stat() will return a localisable
			 * error message
			 */
			return self.failure(Muggle(self.lang[err] || err));
		image.size = rs.stat.size;
		image.dims = [rs.dims.width, rs.dims.height];
		image.hash = rs.hash;
		if (rs.apng)
			image.apng = true;
		self.verified();
	});
};

// Look up binary paths
var identifyBin, convertBin, exiftoolBin, ffmpegBin, pngquantBin;
etc.which('identify', function (bin) { identifyBin = bin; });
etc.which('convert', function (bin) { convertBin = bin; });
if (config.DEL_EXIF)
	etc.which('exiftool', function (bin) { exiftoolBin = bin; });
if (config.WEBM)
	etc.which('ffmpeg', function (bin) { ffmpegBin = bin; });
if (config.PNG_THUMBS)
	etc.which('pngquant', function (bin) { pngquantBin = bin; });

// Flow control and callbacks of stream -> child process jobs
function undine(stream, child, cb) {
	// Hold the response buffers for later concatenation
	var stderr = [],
		stdout = [];
	stream.pipe(child.stdin);
	// Proxy errors to stream
	child.once('error', function(err) {
		stream.emit('error', err);
	});
	child.stderr.on('data', function(data) {
		stderr.push(data);
	});
	child.stdout.on('data', function(data) {
		stdout.push(data);
	});
	child.once('close', function() {
		stderr = stderr.length === 0 ? null : Buffer.concat(stderr);
		cb(stderr, Buffer.concat(stdout));
	});
}

// Pass file to imagemagick's identify
function identify(stream, cb) {
	undine(stream, child_process.spawn(identifyBin, ['-', '-format', '%Wx%H']),
		function(err, out) {
			if (err) {
				var msg = 'im_bad';
				err = err.toString();
				if (err.match(/no such file/i))
					msg = 'im_missing';
				else if (err.match(/improper image header/i))
					msg = 'im_not_image';
				else if (err.match(/no decode delegate/i))
					msg = 'im_unsupported';
				return cb(msg);
			}
			const m = out.toString().trim().match(/(\d+)x(\d+)/);
			if (!m)
				return cb('im_dims_fail');
			else {
				return cb(null, {
					width: parseInt(m[1], 10),
					height: parseInt(m[2], 10)
				});
			}
		}
	);
}

function detectAPNG(stream, cb) {
	var detector = new findapng.apngDetector(),
		done;
	/*
	 * If it throws an exception, that's pretty much it for the server. Don't
	 * know if there is actually a better method of error handling for native
	 * code.
	 */
	stream.on('data', function(data) {
		if (done)
			return;
		const result = detector.Detect(data),
			isAPNG = result === 1;
		if (isAPNG || result === 2) {
			done = true;
			cb(null, isAPNG);
		}
	});
}

/*
 * In-memory image duplicate detection ala findimagedupes.pl.
 * stream -> convert -> buffer
 */
function perceptual_hash(stream, cb) {
	undine(stream, child_process.spawn(convertBin, [
			'-',
			'-background', 'white', '-mosaic', '+matte',
			'-sample', '160x160!',
			'-type', 'grayscale',
			'-blur', '2x2',
			'-normalize',
			'-equalize', '1',
			'-scale', '16x16',
			'-depth', '1',
			'r:-'
		]),
		function(err, out) {
			/*
			 * Let error fall trough silently. identify() can do the detailed
			 * error logging.
			 */
			// Last char is always padding '='
			out = out.toString('base64').slice(0, -1);
			if (out.length !== 43)
				return cb('im_hashing');
			cb(null, out);
		}
	);
}

IU.verified = function() {
	if (this.failed)
		return;
	const desc = this.image.video ? this.lang.im_video : this.lang.im_image,
		w = this.image.dims[0],
		h = this.image.dims[1];
	if (!w || !h)
		return this.failure(Muggle(this.lang.im_bad_dims));
	if (config.IMAGE_PIXELS_MAX && w * h > config.IMAGE_PIXELS_MAX)
		return this.failure(Muggle(this.lang.im_too_many_pixels));
	if (w > config.IMAGE_WIDTH_MAX && h > config.IMAGE_HEIGHT_MAX)
		return this.failure(Muggle(desc + this.lang.im_too_wide_and_tall));
	if (w > config.IMAGE_WIDTH_MAX)
		return this.failure(Muggle(desc + this.lang.im_too_wide));
	if (h > config.IMAGE_HEIGHT_MAX)
		return this.failure(Muggle(desc + this.lang.im_too_tall));

	var self = this;
	// Perform hash comparison against the database
	self.db.check_duplicate(self.image.hash, function(err) {
		if (err)
			return self.failure(err);
		self.sha1();
	});
};

IU.fill_in_specs = function (specs, kind) {
	specs.src = this.tagged_path;
	specs.ext = this.image.ext;
	specs.dest = this.image.path + '_' + kind;
	this.image[kind + '_path'] = specs.dest;
};

IU.sha1 = function(){
	var f = fs.ReadStream(this.image.path),
		sha1sum = crypto.createHash('sha1'),
		self = this;
	f.on('data', function(d){
		sha1sum.update(d);
	});
	f.on('error', function(err){
		self.failure(Muggle(this.lang.sha1 + err));
	});
	f.on('end', function(){
		self.image.SHA1 = sha1sum.digest('hex');
		self.deduped();
	});
};

IU.deduped = function () {
	if (this.failed)
		return;
	var image = this.image,
		specs = get_thumb_specs(image, this.pinky, 1);
	const w = image.dims[0],
		h = image.dims[1];

	/* Determine whether we really need a thumbnail */
	if (image.size < 30*1024
			&& ['.jpg', '.png'].indexOf(image.ext) >= 0
			&& !image.apng && !image.video
			&& w <= specs.dims[0] && h <= specs.dims[1]) {
		return this.got_nails();
	}
	this.fill_in_specs(specs, 'thumb');
	var self = this;
	image.dims = [w, h].concat(specs.dims);
	this.status(this.lang.im_thumbnailing);
	self.resize_and_track(specs, function (err) {
		if (err)
			return self.failure(err);

		if (config.EXTRA_MID_THUMBNAILS)
			self.middle_nail();
		else
			self.got_nails();
	});
};

IU.middle_nail = function () {
	if (this.failed)
		return;

	var specs = get_thumb_specs(this.image, this.pinky, 2);
	this.fill_in_specs(specs, 'mid');

	var self = this;
	this.resize_and_track(specs, function (err) {
		if (err)
			self.failure(err);
		self.got_nails();
	});
};

IU.got_nails = function () {
	if (this.failed)
		return;

	var image = this.image;
	// stop pretending this is a PNG
	if (image.video) {
		image.path = image.video;
		image.ext = image.mp3 ? '.mp3' : '.webm';
		delete image.video;
		delete image.mp3;
	}

	var time = Date.now();
	image.src = time + image.ext;
	var base = path.basename;
	var tmps = {src: base(image.path)};

	if (image.thumb_path) {
		image.thumb = time + '.jpg';
		tmps.thumb = base(image.thumb_path);
	}
	if (image.mid_path) {
		image.mid = time + '.jpg';
		tmps.mid = base(image.mid_path);
	}

	this.record_image(tmps);
};

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

function setup_image_params(o) {
	// only the first time!
	if (o.setup) return;
	o.setup = true;

	o.src += '[0]'; // just the first frame of the animation

	o.dest = o.format + ':' + o.dest;
	o.flatDims = o.dims[0] + 'x' + o.dims[1];
	o.quality += ''; // coerce to string
}

function build_im_args(o) {
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

IU.resize_image = function(o, callback) {
	var args = build_im_args(o);
	// force new size
	args.push('-resize', o.flatDims + '!', '-gamma', '2.2');
	// add background
	if (o.bg)
		args.push('-background', o.bg, '-layers', 'mosaic', '+matte');
	// disregard metadata, acquire artifacts
	args.push('-strip');
	if (o.bg)
		args.push('-quality', o.quality);
	args.push(o.dest);
	var self = this;
	convert(args, o.src, function(err) {
		if (err) {
			winston.warn(err);
			callback(Muggle(self.lang.im_resizing, err));
		}
		// Lossify PNG thumbnail
		else if (!o.bg) {
			var pqDest = o.dest.slice(4);
			child_process.execFile(pngquantBin, [
					'-f', '-o', pqDest,
					'--quality', o.quality,
					pqDest
				],
				function(err) {
					if (err) {
						winston.warn(err);
						callback(Muggle(self.lang.im_pngquant, err));
					} else
						callback(null, o.dest);
				});
		}
		else
			callback(null, o.dest);
	});
};

IU.resize_and_track = function (o, cb) {
	var self = this;
	this.resize_image(o, function (err, fnm) {
		if (err)
			return cb(err);

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
	return files;
}

IU.failure = function (err) {
	var err_desc = this.lang.im_unknown;
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
