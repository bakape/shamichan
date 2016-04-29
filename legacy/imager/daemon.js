/*
 Image and video upload processing
*/

const async = require('async'),
	common = require('../common'),
	config = require('../config'),
	cookie = require('cookie'),
	crypto = require('crypto'),
	child_process = require('child_process'),
	etc = require('../util/etc'),
	{Muggle} = etc,
	db = require('./db'),
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
		this.db = new db.ClientController;
		this.client_id = client_id;
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
		const image_id = common.randomID(32),
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
