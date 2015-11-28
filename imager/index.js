const async = require('async'),
    config = require('../config'),
    child_process = require('child_process'),
    db = require('./db'),
    etc = require('../util/etc'),
    fs = require('fs-extra'),
    hooks = require('../util/hooks'),
    path = require('path'),
    Promise = require('bluebird'),
    winston = require('winston')

exports.ClientController = db.ClientController;
exports.config = config;

const image_attrs = ('src thumb ext dims size MD5 SHA1 hash imgnm spoiler'
	+ ' apng mid audio length').split(' ');
exports.image_attrs = image_attrs;

function nestImageProps(post) {
	if (!is_image(post))
		return;

	// Restructures the flat hash from redis to have image attributes in an
	// embeded hash. Better structure.
	let image = {};
	for (let key of image_attrs) {
		if (key in post) {
			image[key] = post[key];
			delete post[key];
		}
	}
	if (image.dims.split)
		image.dims = image.dims.split(',').map(parse_number);
	image.size = parse_number(image.size);

	// Hashes are only used for image duplicate detection and are useless
	// client-side
	delete image.hash;
	post.image = image;
}
exports.nestImageProps = nestImageProps;

function deleteImageProps(post) {
	if (!is_image(post))
		return;
	for (let key of image_attrs) {
		delete post[key];
	}
}
exports.deleteImageProps = deleteImageProps;

function parse_number(n) {
	return parseInt(n, 10);
}

hooks.hook_sync('inlinePost', function (info) {
	let post = info.dest;
	const image = info.src.image;
	if (!image)
		return;
	for (let i = 0, l = image_attrs.length; i < l; i++) {
		let key = image_attrs[i];
		if (key in image)
			post[key] = image[key];
	}
});

function is_image(image) {
	return image && image.src;
}

function media_path(dir, filename) {
	return path.join(config.MEDIA_DIRS[dir], filename);
}
exports.media_path = media_path;

function squish_MD5 (hash) {
	if (typeof hash == 'string')
		hash = new Buffer(hash, 'hex');
	return hash.toString('base64').replace(/\//g, '_').replace(/=*$/, '');
}
exports.squish_MD5 = squish_MD5;
