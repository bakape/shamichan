var _ = require('./lib/underscore'),
    async = require('async'),
    config = require('./config'),
    fs = require('fs'),
    path = require('path');

_.templateSettings = {
	interpolate: /\{\{(.+?)\}\}/g
};

exports.dbCache = {
	OPs: {},
	opTags: {},
	threadSubs: {},
	YAKUMAN: 0,
	funThread: 0,
};

var RES = exports.resources = {};

exports.reset_resources = function (cb) {
	var deps = config.CLIENT_DEPS;
	function read(file) {
		return fs.readFile.bind(fs, file, 'UTF-8');
	}
	function tmpl(data) {
		return _.template(data, config).split(/\$[A-Z]+/);
	}
	async.parallel({
		version: require('./get').get_version.bind(null, deps),
		index: read('index.html'),
		filter: read('filter.html'),
		notFound: read(path.join(config.MEDIA_DIR, '404.html')),
		modJs: read('mod.js'),
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
