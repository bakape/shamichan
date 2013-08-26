var _ = require('../lib/underscore'),
    async = require('async'),
    config = require('../config'),
    crypto = require('crypto'),
    fs = require('fs'),
    hooks = require('../hooks'),
    path = require('path'),
    pipeline = require('../pipeline'),
    vm = require('vm');

_.templateSettings = {
	interpolate: /\{\{(.+?)\}\}/g
};

exports.emitter = new (require('events').EventEmitter);

exports.dbCache = {
	OPs: {},
	opTags: {},
	threadSubs: {},
	YAKUMAN: 0,
	funThread: 0,
	bannerState: {},
	imageAllocCleanups: {},
	addresses: {},
	ranges: {},
};

var HOT = exports.hot = {};
var RES = exports.resources = {};
exports.clients = {};
exports.clientsByIP = {};

function reload_hot_config(cb) {
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
		read_exits('exits.txt', function () {
			hooks.trigger('reloadHot', HOT, cb);
		});
	});
}

function reload_scripts(cb) {
	var json = path.join('state', 'scripts.json');
	fs.readFile(json, 'UTF-8', function (err, json) {
		if (err)
			cb(err);
		var js;
		try {
			js = JSON.parse(json);
		}
		catch (e) {
			return cb(e);
		}
		if (!js || !js.vendor || !js.client)
			return cb('Bad state/scripts.json.');

		HOT.VENDOR_JS = js.vendor;
		HOT.CLIENT_JS = js.client;

		var modJs = path.join('state', js.mod);
		fs.readFile(modJs, 'UTF-8', function (err, modSrc) {
			if (err)
				return cb(err);
			RES.modJs = modSrc;
			cb(null);
		});
	});
}

function reload_resources(cb) {

	var deps = require('../deps');

	function read(dir, file) {
		return fs.readFile.bind(fs, path.join(dir, file), 'UTF-8');
	}
	function tmpl(data) {
		var templateVars = _.clone(HOT);
		_.extend(templateVars, require('../imager/config'));
		_.extend(templateVars, config);
		var expanded = _.template(data, templateVars);
		return {tmpl: expanded.split(/\$[A-Z]+/),
			src: expanded};
	}
	async.parallel({
		index: read('tmpl', 'index.html'),
		filter: read('tmpl', 'filter.html'),
		login: read('tmpl', 'login.html'),
		curfew: read('tmpl', 'curfew.html'),
		suspension: read('tmpl', 'suspension.html'),
		aLookup: read('tmpl', 'alookup.html'),
		notFound: read('www', '404.html'),
		serverError: read('www', '50x.html'),
	}, function (err, res) {
		if (err)
			return cb(err);

		var index = tmpl(res.index);
		RES.indexTmpl = index.tmpl;
		var hash = crypto.createHash('md5').update(index.src);
		RES.indexHash = hash.digest('hex').slice(0, 8);
		RES.navigationHtml = make_navigation_html();

		RES.filterTmpl = tmpl(res.filter).tmpl;
		RES.curfewTmpl = tmpl(res.curfew).tmpl;
		RES.suspensionTmpl = tmpl(res.suspension).tmpl;
		RES.loginHtml = tmpl(res.login).tmpl;
		RES.aLookupHtml = res.aLookup;
		RES.notFoundHtml = res.notFound;
		RES.serverErrorHtml = res.serverError;

		hooks.trigger('reloadResources', RES, cb);
	});
}

exports.reload_hot_resources = function (cb) {
	pipeline.refresh_deps();

	async.series([
		reload_hot_config,
		pipeline.rebuild,
		reload_scripts,
		reload_resources,
	], cb);
}

function make_navigation_html() {
	if (!HOT.INTER_BOARD_NAVIGATION)
		return '';
	var bits = ['<nav>['];
	config.BOARDS.forEach(function (board, i) {
		if (board == config.STAFF_BOARD)
			return;
		if (i > 0)
			bits.push(' / ');
		bits.push('<a href="../'+board+'/">'+board+'</a>');
	});
	bits.push(']</nav>');
	return bits.join('');
}

function read_exits(file, cb) {
	fs.readFile(file, 'UTF-8', function (err, lines) {
		if (err)
			return cb(err);
		var exits = [], dest = HOT.BANS;
		lines.split(/\n/g).forEach(function (line) {
			var m = line.match(/^(?:^#\d)*(\d+\.\d+\.\d+\.\d+)/);
			if (!m)
				return;
			var exit = m[1];
			if (dest.indexOf(exit) < 0)
				dest.push(exit);
		});
		cb(null);
	});
}
