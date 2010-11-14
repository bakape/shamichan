var config = require('./config'),
	db = require('./db'),
	exec = require('child_process').exec,
	flow = require('flow'),
	formidable = require('formidable'),
	fs = require('fs'),
	path = require('path');

exports.readable_filesize = function (size) {
       /* Metric. Deal with it. */
       if (size < 1000)
               return size + ' B';
       if (size < 1000000)
               return Math.round(size / 1000) + ' KB';
       size = Math.round(size / 100000).toString();
       return size.slice(0, -1) + '.' + size.slice(-1) + ' MB';
}

exports.ImageUpload = function (clients, allocate_post, broadcast) {
	this.clients = clients;
	this.allocate_post = allocate_post;
	this.broadcast = broadcast;
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
	image.ext = path.extname(image.filename).toLowerCase();
	if (config.IMAGE_EXTS.indexOf(image.ext) < 0)
		return this.failure('Invalid image format.');
	image.tagged_path = image.ext.replace('.', '') + ':' + image.path;
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
	var image = this.image;
	image.pinky = (this.client.post && this.client.post.op) ||
			(this.alloc && this.alloc.op);
	var dims, quality;
	if (image.pinky) {
		dims = config.PINKY_DIMENSIONS;
		quality = config.PINKY_QUALITY;
	}
	else {
		dims = config.THUMB_DIMENSIONS;
		quality = config.THUMB_QUALITY;
	}

	var self = this;
	flow.exec(function () {
		self.read_image_filesize(this);
	}, function (size) {
		image.size = size;
		self.MD5_image(this);
	}, function (MD5) {
		image.MD5 = MD5;
		self.read_image_dimensions(image.tagged_path, this);
	}, function (w, h) {
		image.dims = [w, h];
		image.thumb_path = image.path + '_thumb';
		self.resize_image(image.thumb_path, dims, quality, this);
	}, function () {
		self.read_image_dimensions(image.thumb_path, this);
	}, function (w, h) {
		image.dims.push(w, h);
		self.publish();
	});
};

IU.read_image_filesize = function (callback) {
	fs.stat(this.image.path, function (err, stat) {
		if (err)
			return this.failure('Internal filesize error.');
		if (stat.size > config.IMAGE_FILESIZE_MAX)
			this.failure('File is too large.');
		else
			callback(stat.size);
	});
};

IU.read_image_dimensions = function (path, callback) {
	exec('identify ' + path, function (error, stdout, stderr) {
		if (error) {
			console.log(stderr);
			return this.failure('Corrupt image.');
		}
		var m = stdout.match(/.* (\d+)x(\d+) /);
		if (!m)
			return this.failure('Corrupt image.');
		callback(parseInt(m[1]), parseInt(m[2]));
	});
};

IU.MD5_image = function (callback) {
	exec('md5sum -b ' + this.image.path, function (error, stdout, stderr) {
		if (error) {
			console.log(stderr);
			return this.failure('Hashing error.');
		}
		callback(stdout.match(/^([\da-f]+)/)[1]);
	});
};

IU.resize_image = function (dest, dims, quality, callback) {
	var path = this.image.tagged_path;
	this.handle_exec('convert ' + path + '[0] -gamma 0.454545 ' +
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
	var image = this.image;
	image.time = new Date().getTime();
	image.src = image.time + image.ext;
	image.thumb = image.time + (image.pinky ? '.jpg' : 'l.jpg');
	var dest = path.join(config.IMAGE_DIR, image.src),
		nail = path.join(config.THUMB_DIR, image.thumb);

	var self = this;
	flow.exec(function () {
		self.handle_exec('mv -- ' + image.path + ' ' + dest,
				"Couldn't publish image.", this);
	}, function () {
		image.path = dest;
		self.handle_exec('mv -- ' + image.thumb_path + ' ' + nail,
				"Couldn't publish thumbnail.", this);
	}, function () {
		image.thumb_path = nail;
		db.insert_image(image, this);
	}, function (err, id) {
		if (err)
			return upload_failure("Couldn't add image to DB.");
		image.id = id;
		var info = {
			src: image.src, thumb: image.thumb,
			name: image.filename, dims: image.dims,
			size: exports.readable_filesize(image.size),
			MD5: image.MD5
		};
		if (self.client.post) {
			self.iframe_call('upload_complete', info);
			self.client.post.image = info;
			self.client.uploading = false;
			self.broadcast(info, self.client);
		}
		else {
			var alloc_func = self.allocate_post;
			info.id = id;
			alloc_func(self.alloc, info, self.client, this);
		}
	}, function (err, a) {
		if (err)
			return this.failure('Bad post.');
		self.client.uploading = false;
		self.iframe_call('postForm.on_allocation', a);
	});
};

IU.iframe_call = function (func, param) {
	var resp = this.resp;
	param = param ? JSON.stringify(param) : '';
	resp.writeHead(200, {'Content-Type': 'text/html; charset=UTF-8'});
	resp.end('<!doctype html>\n<title></title>\n<script>'
		+ 'parent.' + func + '(' + param + ');</script>');
};
