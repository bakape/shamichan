var _ = require('../lib/underscore'),
    async = require('async'),
    config = require('../config'),
    fs = require('fs'),
    get_version = require('../get').get_version,
    path = require('path'),
    vm = require('vm');

_.templateSettings = {
	interpolate: /\{\{(.+?)\}\}/g
};

exports.dbCache = {
	OPs: {},
	opTags: {},
	threadSubs: {},
	sharedConnection: null,
	YAKUMAN: 0,
	funThread: 0,
	bannerState: {},
};

var HOT = exports.hot = {};
var RES = exports.resources = {};

exports.reload_hot = function (cb) {
	fs.readFile('hot.js', 'UTF-8', function (err, js) {
		if (err)
			cb(err);
		var hot = {};
		try {
			vm.runInNewContext(js, hot);
		}
		catch (e) {
			return cb(e);
		}
		if (!hot || !hot.hot)
			return cb('Bad hot config.');

		// Overwrite the original object just in case
		Object.keys(HOT).forEach(function (k) {
			delete HOT[k];
		});
		_.extend(HOT, hot.hot);

		cb(null);
	});
};

function make_dir(base, key, cb) {
	var dir;
	if (base)
		dir = path.join(base, key);
	else
		dir = config.MEDIA_DIRS[key];
	fs.stat(dir, function (err, info) {
		var make = false;
		if (err) {
			if (err.code == 'ENOENT')
				make = true;
			else
				return cb(err);
		}
		else if (!info.isDirectory())
			return cb(dir + " is not a directory");
		if (make)
			fs.mkdir(dir, cb);
		else
			cb(null);
	});
}

exports.make_media_dirs = function (cb) {
	var keys = ['src', 'thumb', 'vint', 'dead'];
	async.forEach(keys, make_dir.bind(null, null), function (err) {
		if (err)
			return cb(err);
		var dead = config.MEDIA_DIRS.dead;
		async.forEach(['src', 'thumb'], make_dir.bind(null, dead), cb);
	});
}

exports.reset_resources = function (cb) {
	var deps = require('../deps');
	function read(dir, file) {
		return fs.readFile.bind(fs, path.join(dir, file), 'UTF-8');
	}
	function tmpl(data) {
		return _.template(data, config).split(/\$[A-Z]+/);
	}
	async.parallel({
		version: get_version.bind(null, deps.CLIENT_DEPS),
		index: read('tmpl', 'index.html'),
		filter: read('tmpl', 'filter.html'),
		notFound: read('www', '404.html'),
		modJs: read('client', 'mod.js'),
	}, function (err, res) {
		if (err)
			return cb(err);
		if (config.DEBUG)
			config.CLIENT_JS = 'client.debug.js';
		else
			config.CLIENT_JS = 'client-' + res.version + '.js';
		RES.indexTmpl = tmpl(res.index);
		RES.filterTmpl = tmpl(res.filter);
		RES.notFoundHtml = res.notFound;
		RES.modJs = res.modJs;
		cb(null);
	});
};
