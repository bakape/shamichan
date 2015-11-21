/*
 Image and video upload processing
*/

const async = require('async'),
	common = require('../common/index'),
	config = require('../config'),
	cookie = require('cookie'),
	crypto = require('crypto'),
	child_process = require('child_process'),
	etc = require('../util/etc'),
	{Muggle} = etc,
	imagerDb = require('./db'),
	index = require('./'),
	findapng = require('bindings')('findapng'),
	formidable = require('formidable'),
	fs = require('fs'),
	jobs = require('./jobs'),
	lang = require('../lang'),
	path = require('path'),
	urlParse = require('url').parse,
	util = require('util'),
	winston = require('winston');

const IMAGE_EXTS = ['.png', '.jpg', '.gif'];
for (let ext of ['webm', 'mp3', 'svg', 'pdf']) {
	if (config[ext.toUpperCase()])
		IMAGE_EXTS.push('.' + ext);
}

function new_upload(req, resp) {
	const upload = new ImageUpload(req, resp);
	upload.handle_request(req, resp);
}
exports.new_upload = new_upload;

class ImageUpload {
	constructor(client_id) {
		this.db = new imagerDb.Onegai;
		this.client_id = client_id;
	}
	status(msg) {
		this.client_call('status', msg);
	}
	client_call (t, msg) {
		this.db.client_message(this.client_id, {t, arg: msg});
	}
	respond(code, msg) {
		if (!this.resp)
			return;
		this.resp.writeHead(code, {
			'Content-Type': 'text/html; charset=UTF-8',
			'Access-Control-Allow-Origin': config.MAIN_SERVER_ORIGIN
		});
		this.resp.end(common.parseHTML
			`<!doctype html>
			<title>
				Upload result
			</title>
			This is a legitimate imager response.
			<script>
				parent.postMessage(${JSON.stringify(msg) + ', '
					+ JSON.stringify(config.MAIN_SERVER_ORIGIN)
				});
			</script>`);
		this.resp = null;
	}
	handle_request(req, resp) {
		if (req.method.toLowerCase() != 'post') {
			resp.writeHead(405, {Allow: 'POST'});
			resp.end();
			return;
		}
		this.resp = resp;
		const query = req.query || urlParse(req.url, true).query;

		// Set response language. Checks if client language is set and exixts on
		// the server.
		this.lang = lang[etc.resolveConfig(config.LANGS,
			cookie.parse(req.headers.cookie).lang, config.DEFAULT_LANG)].im;

		this.client_id = parseInt(query.id, 10);
		if (!this.client_id || this.client_id < 1) {
			this.respond(400, this.lang.bad_client);
			return;
		}

		const len = parseInt(req.headers['content-length'], 10);
		if (len > 0 && len > config.IMAGE_FILESIZE_MAX + (20*1024))
			return this.failure(Muggle(this.lang.too_large));

		const form = new formidable.IncomingForm();
		form.uploadDir = config.MEDIA_DIRS.tmp;
		form.maxFieldsSize = 50 * 1024;
		form.hash = 'md5';
		form.onPart = function (part) {
			if (part.filename && part.name == 'image')
				form.handlePart(part);
			else if (!part.filename && ['spoiler', 'op'].indexOf(part.name) >= 0)
				form.handlePart(part);
		};
		form.once('error', err =>
			this.failure(Muggle(this.lang.req_problem, err)));
		form.once('aborted', err =>
			this.failure(Muggle(this.lang.aborted, err)));
		this.lastProgress = 0;
		form.on('progress', this.upload_progress_status.bind(this));

		try {
			form.parse(req, this.parse_form.bind(this));
		}
		catch (err) {
			this.failure(err);
		}
	}
	upload_progress_status(received, total) {
		const percent = Math.floor(100 * received / total),
			increment = (total > (512 * 1024)) ? 10 : 25,
			quantized = Math.floor(percent / increment) * increment;
		if (quantized > this.lastProgress) {
			this.status(percent + this.lang.received);
			this.lastProgress = quantized;
		}
	}
	parse_form(err, fields, files) {
		if (err)
			return this.failure(Muggle(this.lang.invalid, err));
		if (!files.image)
			return this.failure(Muggle(this.lang.no_image));
		this.image = files.image;
		this.pinky = !!parseInt(fields.op, 10);

		const spoiler = parseInt(fields.spoiler, 10);
		if (spoiler) {
			if (config.SPOILER_IMAGES.indexOf(spoiler) < 0)
				return this.failure(Muggle(this.lang.bad_spoiler));
			this.image.spoiler = spoiler;
		}

		this.image.MD5 = index.squish_MD5(this.image.hash);
		this.image.hash = null;

		this.db.track_temporary(this.image.path, err => {
			if (err)
				winston.warn(this.lang.temp_tracking + err);
			this.process();
		});
	}
	process() {
		if (this.failed)
			return;
		const {image} = this,
			filename = image.filename || image.name;
		image.ext = path.extname(filename).toLowerCase();
		if (image.ext == '.jpeg')
			image.ext = '.jpg';
		if (IMAGE_EXTS.indexOf(image.ext) < 0)
			return this.failure(Muggle(this.lang.invalid_format));
		image.imgnm = filename.substr(0, 256);

		this.status(this.lang.verifying);
		if (image.ext == '.webm' || image.ext == '.mp3')
			video_still(image.path, this.verify_webm.bind(this));
		else
			this.verify_image();
	}
	verify_webm(err, info) {
		if (err)
			return this.failure(Muggle(this.lang[err] || err));
		this.db.track_temporary(info.still_path, err => {
			if (err)
				winston.warn("Tracking error: " + err);

			if (info.audio && !config.WEBM_AUDIO)
				return this.failure(Muggle(this.lang.audio_kinshi));

			// pretend it's a PNG for the next steps
			const {image} = this;
			image.video = image.path;
			image.path = info.still_path;
			image.ext = '.png';
			for (let prop of ['audio', 'length', 'mp3']) {
				if (info[prop])
					image[prop] = info[prop];
			}

			this.verify_image();
		});
	}
	verify_image() {
		const {image} = this,
			stream = fs.createReadStream(image.path);
		this.tagged_path = image.ext.replace('.', '') + ':' + image.path;

		stream.once('err', err => {
			winston.error(err);
			stream.close();
			this.failure(Muggle(err));
		});
		const checks = {
			// Get more accurate filesize. Formidable passes the gzipped one
			stat: fs.stat.bind(fs, image.video || image.path),
			dims: this.identify.bind(this, stream),
			hash: this.perceptual_hash.bind(this, stream)
		};
		if (image.ext == '.png')
			checks.apng = this.detectAPNG.bind(this, stream);

		async.parallel(checks, (err, rs) => {
			if (err)
				/*
				 * All functions, except for fs.stat() will return a localisable
				 * error message
				 */
				return this.failure(Muggle(this.lang[err] || err));
			image.size = rs.stat.size;
			image.dims = [rs.dims.width, rs.dims.height];
			image.hash = rs.hash;
			if (rs.apng)
				image.apng = true;
			this.verified();
		});
	}
	// Flow control and callbacks of stream -> child process -> buffer jobs
	undine(stream, child, cb) {
		// Hold the response buffers for later concatenation
		const stderr = [],
			stdout = [];
		stream.pipe(child.stdin);
		// Proxy errors to stream
		child.once('error', err =>
			stream.emit('error', err));
		child.stderr.on('data', data =>
			stderr.push(data));
		child.stdout.on('data', data =>
			stdout.push(data));
		child.once('close', () => {
			const err = stderr.length === 0 ? null : Buffer.concat(stderr);
			cb(err, Buffer.concat(stdout));
		});
	}
	// Pass file to imagemagick's identify
	identify(stream, cb) {
		const child = child_process.spawn(identifyBin, [
			'-[0]', '-format', '%Wx%H'
		]);
		this.undine(stream, child, function(err, out) {
			if (err) {
				let msg = 'bad';
				err = err.toString();
				if (err.match(/no such file/i))
					msg = 'missing';
				else if (err.match(/improper image header/i))
					msg = 'not_image';
				else if (err.match(/no decode delegate/i))
					msg = 'unsupported';
				return cb(msg);
			}
			const m = out.toString().trim().match(/ (\d+)x(\d+)/);
			if (!m)
				return cb('dims_fail');
			return cb(null, {
				width: parseInt(m[1], 10),
				height: parseInt(m[2], 10)
			});
		});
	}
	// In-memory image duplicate detection ala findimagedupes.pl.
	perceptual_hash(stream, cb) {
		const child = child_process.spawn(convertBin, [
			'-[0]',
			'-background', 'white', '-mosaic', '+matte',
			'-sample', '160x160!',
			'-type', 'grayscale',
			'-blur', '2x2',
			'-normalize',
			'-equalize', '1',
			'-scale', '16x16',
			'-depth', '1',
			'r:-'
		]);
		this.undine(stream, child, function(err, out) {
			/*
			 * Let error fall trough silently. identify() can do the detailed
			 * error logging.
			 */
			// Last char is always padding '='
			out = out.toString('base64').slice(0, -1);
			if (out.length !== 43)
				return cb('hashing');
			cb(null, out);
		});
	}
	detectAPNG(stream, cb) {
		const detector = new findapng.apngDetector();
		let done;
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
	verified() {
		if (this.failed)
			return;
		const {image, lang} = this,
			desc = lang[image.video ? 'video' : 'image'],
			[w, h] = image.dims;
		if (!w || !h)
			return this.failure(Muggle(lang.bad_dims));
		if (config.IMAGE_PIXELS_MAX && w * h > config.IMAGE_PIXELS_MAX)
			return this.failure(Muggle(lang.too_many_pixels));
		if (w > config.IMAGE_WIDTH_MAX && h > config.IMAGE_HEIGHT_MAX)
			return this.failure(Muggle(desc + lang.too_wide_and_tall));
		if (w > config.IMAGE_WIDTH_MAX)
			return this.failure(Muggle(desc + lang.too_wide));
		if (h > config.IMAGE_HEIGHT_MAX)
			return this.failure(Muggle(desc + lang.too_tall));

		// Perform hash comparison against the database
		this.db.check_duplicate(image.hash, err => {
			if (err)
				return this.failure(err);
			this.deduped();
		});
	}
	// Start the thumbnailing pathway
	deduped() {
		if (this.failed)
			return;
		const {image} = this,
			specs = this.get_thumb_specs(image, this.pinky, 1);
		let noThumbs;
		const [w, h] = image.dims;

		/* Determine whether we really need a thumbnail */
		if (image.size < 30*1024
			&& ['.jpg', '.png'].indexOf(image.ext) >= 0
			&& !image.apng && !image.video
			&& w <= specs.dims[0]
			&& h <= specs.dims[1]
		)
			noThumbs = true;

		const stream = fs.createReadStream(image.path);
		stream.once('error', err =>
			this.failure(Muggle(this.lang.resizing, err)));

		const pipes = {sha1: this.sha1Hash.bind(this, stream)};
		image.dims = [w, h].concat(specs.dims);
		if (!noThumbs) {
			this.status(this.lang.thumbnailing);
			this.fill_in_specs(specs, 'thumb');

			// Default 125x125 thumbnail
			pipes.thumb = cb => this.resize_image(specs, stream, cb);

			// Extra 250x250 thumbnail
			if (config.EXTRA_MID_THUMBNAILS) {
				pipes.mid = cb => {
					const specs = this.get_thumb_specs(image, this.pinky, 2);
					this.fill_in_specs(specs, 'mid');
					this.resize_image(specs, stream, cb);
				}
			}
		}

		async.parallel(pipes, (err, res) => {
			if (err)
				return this.failure(err);
			this.image.SHA1 = res.sha1;
			this.got_nails();
		});
	}
	/*
	 * Currently only used for exhentai image search, but might as well do it
	 * for everything for consistency.
	 */
	sha1Hash(stream, cb){
		const sha1sum = crypto.createHash('sha1');
		stream.on('data', data =>
			sha1sum.update(data));
		stream.once('end', () =>
			cb(null, sha1sum.digest('hex')));
	}
	get_thumb_specs(image, pinky, scale) {
		const [w, h] = image.dims,
			bound = config[pinky ? 'PINKY_DIMENSIONS' : 'THUMB_DIMENSIONS'],
			r = Math.max(w / bound[0], h / bound[1], 1),
			dims = [Math.round(w/r) * scale, Math.round(h/r) * scale];
		var specs = {bound, dims, format: 'jpg'};

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
	fill_in_specs(specs, kind) {
		specs.src = this.tagged_path;
		specs.ext = this.image.ext;
		specs.dest = this.image.path + '_' + kind;
		this.image[kind + '_path'] = specs.dest;
	}
	resize_image(o, stream, cb) {
		const args = this.build_args(o);
		// force new size
		args.push('-resize', o.flatDims + '!', '-gamma', '2.2');
		// add background
		if (o.bg)
			args.push('-background', o.bg, '-layers', 'mosaic', '+matte');
		// disregard metadata, acquire artifacts
		args.push('-strip');
		// o.quality differs for PNG and non-PNG thumbs
		if (o.bg)
			args.push('-quality', o.quality);
		args.push(o.dest);

		const convert = child_process.spawn(convertBin, args),
			stderr = [];
		stream.pipe(convert.stdin);
		convert.once('error', err => stream.emit('error', err));
		convert.stderr.on('data', data => stderr.push(data));

		// Lossy PNG compression
		if (!o.bg)
			return this.pngquant(stream, convert.stdout, o.fnm, o.quality, cb);

		convert.once('close', () => {
			if (stderr.length !== 0) {
				const err = Buffer.concat(stderr).toString();
				winston.warn(err);
				return cb(Muggle(this.lang.resizing, err));
			}
			this.db.track_temporary(o.fnm, cb);
		});
	}
	build_args(o) {
		// avoid OOM killer
		const args = ['-limit', 'memory', '32', '-limit', 'map', '64'],
			{dims} = o;

		// resample from twice the thumbnail size
		// (avoid sampling from the entirety of enormous 6000x6000 images etc)
		const samp = dims[0]*2 + 'x' + dims[1]*2;
		if (o.ext == '.jpg')
			args.push('-define', 'jpeg:size=' + samp);
		this.setup_image_params(o);
		// Process only the first frame from stdin
		args.push('-[0]');
		if (o.ext != '.jpg')
			args.push('-sample', samp);
		// gamma-correct yet shitty downsampling
		args.push('-gamma', '0.454545', '-filter', 'box');
		return args;
	}
	setup_image_params(o) {
		// only the first time!
		if (o.setup)
			return;
		o.setup = true;

		// Store image filename without imagemagick prefix
		o.fnm = o.dest;

		/*
		 * PNG files will also be piped into pngquant, if PNG_THUMBS is enabled.
		 * Otherwise will write straight to disk.
		 */
		o.dest = o.format + ':' + (o.bg ? o.dest : '-');
		o.flatDims = o.dims[0] + 'x' + o.dims[1];
		o.quality += ''; // coerce to string
	}
	// Lossy PNG thumbnail compression
	pngquant(stream, out, dest, quality, cb) {
		const child = child_process.spawn(pngquantBin, [
			'--quality', quality, '-'
		]);
		const stderr = [];
		out.pipe(child.stdin);

		/*
		 * pngquant can only pipe to sdtdout, when piped in through stdin.
		 * Retarded, I know. So we need to pipe it back to node's writestream.
		 */
		child.stdout.pipe(fs.createWriteStream(dest));
		child.once('error', err => stream.emit('error', err));
		child.stderr.on('data', data => stderr.push(data));
		child.once('close', () => {
			if (stderr.length !==0) {
				const err = Buffer.concat(stderr).toString();
				winston.warn(err);
				return cb(Muggle(this.lang.pngquant, err));
			}

			// PNG thumbnails generated
			this.image.png_thumbs = true;
			this.db.track_temporary(dest, cb);
		});
	}
	got_nails() {
		if (this.failed)
			return;

		const {image} = this;
		// stop pretending this is a PNG
		if (image.video) {
			image.path = image.video;
			image.ext = image.mp3 ? '.mp3' : '.webm';
			delete image.video;
			delete image.mp3;
		}

		const time = Date.now();
		image.src = time + image.ext;
		const tmps = {src: path.basename(image.path)},
			// Thumbnail extension
			ext = image.png_thumbs ? '.png' : '.jpg';

		for (let type of ['thumb', 'mid']) {
			const imagePath = image[type + '_path'];
			if (imagePath) {
				image[type] = time + ext;
				tmps[type] = path.basename(imagePath);
			}
		}

		this.record_image(tmps);
	}
	record_image(tmps) {
		if (this.failed)
			return;
		const view = {};
		for (let attr of index.image_attrs) {
			if (attr)
				view[attr] = this.image[attr]
		}
		view.pinky = this.pinky;
		const image_id = common.random_id().toFixed(),
			alloc = {image: view, tmps};
		this.db.record_image_alloc(image_id, alloc, err => {
			if (err)
				return this.failure("Image storage failure.");
			this.client_call('alloc', image_id);
			this.respond(202, 'OK');
		});
	}
	failure(err) {
		let err_desc = this.lang.unknown;
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
			const files = this.image_files(this.image);
			files.forEach(function (file) {
				fs.unlink(file, function (err) {
					if (err)
						winston.warn(`Deleting ${file}: ${err}`);
				});
			});
			this.db.lose_temporaries(files, function (err) {
				if (err)
					winston.warn("Tracking failure: " + err);
			});
		}
	}
	image_files(image) {
		const files = [];
		if (image.path)
			files.push(image.path);
		for (let type of ['thumb', 'mid']) {
			const path = type + '_path';
			if (image[path])
				files.push(path);
		}
		return files;
	}
}

function StillJob(src) {
	jobs.Job.call(this);
	this.src = src;
}
util.inherits(StillJob, jobs.Job);

StillJob.prototype.describe_job = function () {
	return "FFmpeg video still of " + this.src;
};

StillJob.prototype.perform_job = function () {
	const dest = index.media_path('tmp', 'still_' + common.random_id());
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
			const first = (stderr ? stderr.split('\n') : [])[0],
				is_webm = /matroska,webm/i.test(first),
				isMP3 = /mp3/i.test(first);
			if (err) {
				var msg;
				if (/no such file or directory/i.test(first))
					msg = 'missing';
				else if (/invalid data found when/i.test(first))
					msg = "video_invalid";
				else if (/^ffmpeg version/i.test(first))
					msg = "ffmpeg_too_old";
				else if (isMP3)
					msg = 'mp3_no_cover';
				else {
					msg = "video_unknown";
					winston.warn("Unknown ffmpeg output: " + first);
				}
				fs.unlink(dest, function () {
					self.finish_job(msg);
				});
				return;
			}
			if (!is_webm  && !isMP3) {
				fs.unlink(dest, function () {
					self.finish_job('video_format');
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
				audio: (is_webm && /audio:\s*(?:vorbis|opus)/i.test(stderr))
					|| isMP3,
				length: length,
				mp3: isMP3
			});
		}
	);
};

function video_still(src, cb) {
	jobs.schedule(new StillJob(src), cb);
}

// Look up binary paths
var identifyBin, convertBin, exiftoolBin, ffmpegBin, pngquantBin;
etc.which('identify', function (bin) { identifyBin = bin; });
etc.which('convert', function (bin) { convertBin = bin; });
if (config.PNG_THUMBS)
	etc.which('pngquant', function (bin) { pngquantBin = bin; });
if (config.WEBM)
	etc.which('ffmpeg', function (bin) { ffmpegBin = bin; });
