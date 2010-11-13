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

var validFields = ['client_id', 'alloc'];

exports.handle_upload = function (req, resp, clients, env) {
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
		else {
			this._error('Superfluous field.');
			if (part.path)
				console.log(part.path);
		}
	};
	form.parse(req, function (err, fields, files) {
		if (err) {
			console.log("Upload error: " + err);
			var code = 500;
			err = '' + (err.message || err);
			return client_call(resp, 'upload_error', err);
		}
		var image = files.image;
		image.resp = resp;
		if (!image)
			return client_call(resp, 'upload_error',
					'No image supplied.');
		var client = clients[fields.client_id];
		if (!client)
			return upload_failure(image, 'Invalid client id.');
		image.client = client;
		if (client.uploading) {
			upload_failure(image, 'Already uploading.');
			client.uploading = true;
			return;
		}
		client.uploading = true;
		if (client.post && client.post.image)
			return upload_failure(image, 'Image already exists.');
		image.ext = path.extname(image.filename).toLowerCase();
		if (config.IMAGE_EXTS.indexOf(image.ext) < 0)
			return upload_failure(image, 'Invalid image format.');
		image.tagged_path = image.ext.replace('.', '') +
				':' + image.path;
		if (fields.alloc) {
			try {
				image.alloc = JSON.parse(fields.alloc);
			}
			catch (e) {
				return upload_failure(image, 'Bad alloc.');
			}
			if (client.post)
				return upload_failure(image,'Existing alloc.');
		}
		else if (!client.post)
			return upload_failure(image, 'Missing alloc.');

		var pinky = (client.post && client.post.op
				) || (image.alloc && image.alloc.op);
		var dims, quality;
		if (pinky) {
			dims = config.PINKY_DIMENSIONS;
			quality = config.PINKY_QUALITY;
		}
		else {
			dims = config.THUMB_DIMENSIONS;
			quality = config.THUMB_QUALITY;
		}

		flow.exec(function () {
			read_image_filesize(image, this);
		}, function (size) {
			image.size = size;
			md5_image(image, this);
		}, function (MD5) {
			image.MD5 = MD5;
			read_image_dimensions(image, image.tagged_path, this);
		}, function (w, h) {
			image.dims = [w, h];
			image.thumb_path = image.path + '_thumb';
			resize_image(image, image.thumb_path, dims, quality,
					this);
		}, function () {
			read_image_dimensions(image, image.thumb_path, this);
		}, function (w, h) {
			image.dims.push(w); image.dims.push(h);
			publish_image(image, pinky, env);
		});
	});
}

function read_image_filesize(image, callback) {
	fs.stat(image.path, function (err, stat) {
		if (err) {
			upload_failure(image, 'Internal upload error.');
			return;
		}
		if (stat.size > config.IMAGE_FILESIZE_MAX)
			upload_failure(image, 'File is too large.');
		else
			callback(stat.size);
	});
}

function read_image_dimensions(image, path, callback) {
	exec('identify ' + path, function (error, stdout, stderr) {
		if (error) {
			console.log(stderr);
			return upload_failure(image, 'Corrupt image.');
		}
		var m = stdout.match(/.* (\d+)x(\d+) /);
		if (!m)
			return upload_failure(image, 'Corrupt image.');
		callback(parseInt(m[1]), parseInt(m[2]));
	});
}

function md5_image(image, callback) {
	exec('md5sum -b ' + image.path, function (error, stdout, stderr) {
		if (error) {
			console.log(stderr);
			return upload_failure(image, 'Hashing error.');
		}
		callback(stdout.match(/^([\da-f]+)/)[1]);
	});
}

function resize_image(image, dest, dims, quality, callback) {
	var path = image.tagged_path;
	exec('convert ' + path + '[0] -gamma 0.454545 -filter lanczos -resize '
		+ dims + ' -gamma 2.2 -quality ' + quality + ' jpg:' + dest,
		exec_handler(image, 'Conversion error.', callback));
}

function exec_handler(image, err_desc, callback) {
	return function (error, stdout, stderr) {
		if (error != null)
			console.log(error);
		if (stdout)
			console.log(stdout);
		if (stderr)
			console.log(stderr);
		if (error == null && !stderr)
			callback();
		else
			upload_failure(image, err_desc);
	};
}

function upload_failure(image, err_desc) {
	client_call(image.resp, 'upload_error', err_desc);
	if (image.path)
		fs.unlink(image.path);
	if (image.thumb_path)
		fs.unlink(image.thumb_path);
	if (image.id) {
		/* TODO: Remove DB row */
	}
	image.client.uploading = false;
}

function publish_image(image, pinky, env) {
	image.time = new Date().getTime();
	image.src = image.time + image.ext;
	image.thumb = image.time + (pinky ? '.jpg' : 'l.jpg');
	var dest = path.join(config.IMAGE_DIR, image.src),
		nail = path.join(config.THUMB_DIR, image.thumb);

	flow.exec(function () {
		exec('mv -- ' + image.path + ' ' + dest, exec_handler(
				image, "Couldn't publish image.", this));
	}, function () {
		image.path = dest;
		exec('mv -- ' + image.thumb_path + ' ' + nail, exec_handler(
				image, "Couldn't publish thumbnail.", this));
	}, function () {
		image.thumb_path = nail;
		db.insert_image(image, pinky, this);
	}, function (err, id) {
		if (err)
			return upload_failure("Couldn't add image to DB.");
		image.id = id;
		var info = {
			src: image.src, thumb: image.thumb,
			name: image.filename, dims: image.dims,
			size: exports.readable_filesize(image.size),
			MD5: image.MD5, id: image.id
		};
		var client = image.client;
		if (client.post) {
			client_call(image.resp, 'upload_complete', info);
			client.post.image = info;
			client.uploading = false;
		}
		else
			env.allocate_post(image.alloc, info, client, this);
	}, function (err, a) {
		if (err)
			return upload_failure(image, 'Bad post.');
		image.client.uploading = false;
		client_call(image.resp, 'postForm.on_allocation', a);
	});
}

function client_call(resp, func, param) {
	param = param ? JSON.stringify(param) : '';
	resp.writeHead(200, {'Content-Type': 'text/html; charset=UTF-8'});
	resp.end('<!doctype html>\n<title></title>\n<script>'
		+ 'parent.' + func + '(' + param + ');</script>');
}
