var config = require('./config'),
	db = require('./db'),
	exec = require('child_process').exec,
	flow = require('flow'),
	formidable = require('formidable'),
	fs = require('fs'),
	path = require('path'),
	util = require('util');

var IMAGE_EXTS = ['.png', '.jpg', '.gif'];

function readable_filesize(size) {
       /* Metric. Deal with it. */
       if (size < 1000)
               return size + ' B';
       if (size < 1000000)
               return Math.round(size / 1000) + ' KB';
       size = Math.round(size / 100000).toString();
       return size.slice(0, -1) + '.' + size.slice(-1) + ' MB';
}

function get_thumb_specs(pinky) {
	if (pinky)
		return {dims: config.PINKY_DIMENSIONS,
				quality: config.PINKY_QUALITY, ext: '.jpg'}
	return {dims: config.THUMB_DIMENSIONS, quality: config.THUMB_QUALITY,
			ext: 'l.jpg'};
}

exports.get_image_view = function (image, imgnm, pinky) {
	if (!image.cache) {
		var d = image.dims;
		image.cache = {src: '' + image.time + IMAGE_EXTS[image.ext],
			size: readable_filesize(image.size),
			thumb_dims: d.slice(0, 4),
			pinky_dims: [d[0], d[1], d[4], d[5]]};
	}
	return {src: image.cache.src, size: image.cache.size, MD5: image.MD5,
		thumb: image.time + (pinky ? '.jpg' : 'l.jpg'), imgnm: imgnm,
		dims: pinky ? image.cache.pinky_dims : image.cache.thumb_dims};
};

exports.ImageUpload = function (clients, allocate_post, broadcast, status) {
	this.clients = clients;
	this.allocate_post = allocate_post;
	this.broadcast = broadcast;
	this.status = status;
};

var IU = exports.ImageUpload.prototype;

var validFields = ['client_id', 'alloc'];

IU.handle_request = function (req, resp) {
	this.resp = resp;
	if (!config.IMAGE_UPLOAD) {
		resp.writeHead(403, {'Content-Type': 'text/plain'});
		resp.end('No upload.');
		return;
	}
	var form = new formidable.IncomingForm();
	form.maxFieldsSize = 512;
	form.onPart = function (part) {
		if (part.filename && part.name == 'image')
			form.handlePart(part);
		else if (!part.filename && validFields.indexOf(part.name) >= 0)
			form.handlePart(part);
		else
			this._error('Superfluous field.');
	};
	form.parse(req, this.parse_form.bind(this));
};

IU.parse_form = function (err, fields, files) {
	if (err) {
		console.log("Upload error: " + err);
		var code = 500;
		err = '' + (err.message || err);
		return this.failure(err);
	}

	var image = files.image;
	if (!image)
		return this.failure('No image.');
	this.image = image;
	var client = this.clients[fields.client_id];
	if (!client)
		return this.failure('Invalid client id.');
	this.client = client;

	if (client.uploading) {
		this.failure('Already uploading.');
		/* previous line negated client.uploading, so restore it */
		client.uploading = true;
		return;
	}
	client.uploading = true;
	if (client.post && client.post.image)
		return this.failure('Image already exists.');
	var ext = path.extname(image.filename).toLowerCase();
	if (ext == '.jpeg')
		ext = '.jpg';
	image.tagged_path = ext.replace('.', '') + ':' + image.path;
	image.ext = IMAGE_EXTS.indexOf(ext);
	if (image.ext < 0)
		return this.failure('Invalid image format.');
	if (fields.alloc) {
		try {
			this.alloc = JSON.parse(fields.alloc);
		}
		catch (e) {
			return this.failure('Bad alloc.');
		}
		if (client.post)
			return this.failure('Existing alloc.');
	}
	else if (!client.post)
		return this.failure('Missing alloc.');
	this.process();
}

IU.process = function () {
	this.status('Verifying...');
	var image = this.image;
	image.pinky = (this.client.post && this.client.post.op) ||
			(this.alloc && this.alloc.op);

	var specs = get_thumb_specs(image.pinky);
	var self = this;
	flow.exec(function () {
		self.MD5_image(this);
	}, function (MD5) {
		image.MD5 = MD5;
		db.check_duplicate_image(MD5, this);
	}, function (err, found) {
		if (err)
			return self.failure('Duplicate image check failed.');
		if (found) {
			self.image = found;
			self.image.filename = image.filename;
			return self.adapt_existing(image.pinky);
		}
		self.read_image_filesize(this);
	}, function (size) {
		image.size = size;
		self.read_image_dimensions(image.tagged_path, this);
	}, function (w, h) {
		image.dims = [w, h];
		image.thumb_path = image.path + '_thumb';
		self.status('Thumbnailing...');
		self.resize_image(image.tagged_path, image.thumb_path,
				specs.dims, specs.quality, this);
	}, function () {
		self.read_image_dimensions(image.thumb_path, this);
	}, function (w, h) {
		if (image.pinky)
			image.dims.push(null, null, w, h);
		else
			image.dims.push(w, h, null, null);
		self.status('Publishing...');
		image.time = new Date().getTime();
		image.src = image.time + IMAGE_EXTS[image.ext];
		self.dest = path.join(config.IMAGE_DIR, image.src);
		self.handle_exec('mv -- ' + image.path + ' ' + self.dest,
				"Couldn't publish image.", this);
	}, function () {
		image.path = self.dest;
		image.thumb = image.time + specs.ext;
		self.nail = path.join(config.THUMB_DIR, image.thumb);
		self.handle_exec('mv -- ' + image.thumb_path + ' ' + self.nail,
				"Couldn't publish thumbnail.", this);
	}, function () {
		image.thumb_path = self.nail;
		db.insert_image(image, this);
	}, function (err, id) {
		if (err)
			return upload_failure("Couldn't add image to DB.");
		image.id = id;
		self.publish();
	});
};

IU.adapt_existing = function (pinky) {
	var image = this.image;
	var index = pinky ? 4 : 2;
	var specs = get_thumb_specs(pinky);
	image.src = image.time + IMAGE_EXTS[image.ext];
	image.thumb = image.time + specs.ext;
	if (image.dims[index] !== null) {
		this.status('Publishing...');
		return this.publish();
	}
	image.thumb_path = path.join(config.THUMB_DIR, image.thumb);
	var self = this;
	flow.exec(function () {
		self.status('Thumbnailing...');
		/* Don't set image.src since we don't want to delete it
		 * on failure as it is shared */
		self.resize_image(path.join(config.IMAGE_DIR, image.src),
			image.thumb_path, specs.dims, specs.quality, this);
	}, function () {
		self.read_image_dimensions(image.thumb_path, this);
	}, function (w, h) {
		image.dims[index] = w;
		image.dims[index + 1] = h;
		self.status('Publishing...');
		db.update_thumbnail_dimensions(image.id, pinky, w, h, this);
	}, function (err) {
		if (err)
			return self.failure("Secondary thumbnail failure.");
		self.publish();
	});

};

IU.read_image_filesize = function (callback) {
	var self = this;
	fs.stat(this.image.path, function (err, stat) {
		if (err)
			return self.failure('Internal filesize error.');
		if (stat.size > config.IMAGE_FILESIZE_MAX)
			self.failure('File is too large.');
		else
			callback(stat.size);
	});
};

IU.read_image_dimensions = function (path, callback) {
	var self = this;
	exec('identify ' + path, function (error, stdout, stderr) {
		if (error) {
			console.log(stderr);
			return self.failure('Corrupt image.');
		}
		var m = stdout.match(/.* (\d+)x(\d+) /);
		if (!m)
			return self.failure('Corrupt image.');
		callback(parseInt(m[1]), parseInt(m[2]));
	});
};

IU.MD5_image = function (callback) {
	var self = this;
	exec('md5sum -b ' + this.image.path, function (error, stdout, stderr) {
		if (error) {
			console.log(stderr);
			return self.failure('Hashing error.');
		}
		callback(stdout.match(/^([\da-f]+)/)[1]);
	});
};

IU.resize_image = function (src, dest, dims, quality, callback) {
	this.handle_exec('convert ' + src + '[0] -gamma 0.454545 ' +
			'-filter lanczos -resize ' + dims + ' -gamma 2.2 ' +
			'-quality ' + quality + ' jpg:' + dest,
		'Conversion error.', callback);
};

IU.handle_exec = function (cmd, err_desc, callback) {
	var self = this;
	exec(cmd, function (error, stdout, stderr) {
		if (stdout)
			console.log(stdout);
		if (stderr)
			util.error(stderr);
		if (error == null && !stderr)
			callback();
		else
			self.failure(err_desc);
	});
};

IU.failure = function (err_desc) {
	this.iframe_call('upload_error', err_desc);
	var image = this.image;
	if (image) {
		if (image.path)
			fs.unlink(image.path);
		if (image.thumb_path)
			fs.unlink(image.thumb_path);
		if (image.id) {
			/* TODO: Remove DB row */
		}
	}
	this.client.uploading = false;
};

IU.publish = function () {
	var imgnm = this.image.filename.substr(0, 256);
	var image = {
		time: this.image.time, dims: this.image.dims,
		size: this.image.size, MD5: this.image.MD5,
		id: this.image.id, ext: this.image.ext
	};
	if (this.client.post) {
		var view = exports.get_image_view(image, imgnm,
				!this.image.pinky);
		this.iframe_call('upload_complete', view);
		this.client.post.image = image;
		this.client.post.imgnm = imgnm;
		this.client.uploading = false;
		this.broadcast(view, this.client);
	}
	else {
		var self = this;
		this.allocate_post(this.alloc, image, imgnm, this.client,
				function (err, a) {
			if (err)
				return self.failure('Bad post.');
			self.client.uploading = false;
			self.iframe_call('postForm.on_allocation', a);
		});
	}
};

IU.iframe_call = function (func, param) {
	var resp = this.resp;
	param = param ? JSON.stringify(param) : '';
	resp.writeHead(200, {'Content-Type': 'text/html; charset=UTF-8'});
	resp.end('<!doctype html>\n<title></title>\n<script>'
		+ 'parent.' + func + '(' + param + ');</script>');
};
