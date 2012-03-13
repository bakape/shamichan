var twitter = require('./twitter'),
    url_parse = require('url').parse,
    util = require('util');

var escape = require('../common').escape_html;
var routes = [];

var server = require('http').createServer(function (req, resp) {
	var method = req.method.toLowerCase(), numRoutes = routes.length;
	var parsed = url_parse(req.url, true);
	req.url = parsed.pathname;
	req.query = parsed.query;
	for (var i = 0; i < numRoutes; i++) {
		var route = routes[i];
		if (method != route.method)
			continue;
		var m = req.url.match(route.pattern);
		if (m) {
			route.handler(req, resp, m);
			return;
		}
	}
	if (debug_static.enabled)
		debug_static(req, resp);
	else
		render_404(resp);
});
exports.server = server;

exports.route_get = function (pattern, handler) {
	routes.push({method: 'get', pattern: pattern,
			handler: auth_passthrough.bind(null, handler)});
};

function auth_passthrough(handler, req, resp, params) {
	var chunks = twitter.extract_cookie(req.headers.cookie);
	if (!chunks) {
		handler(req, resp, params);
		return;
	}

	twitter.check_cookie(chunks, false, function (err, auth) {
		if (!err)
			req.auth = auth;
		handler(req, resp, params);
	});
}

exports.route_get_auth = function (pattern, handler) {
	routes.push({method: 'get', pattern: pattern,
			handler: auth_checker.bind(null, handler, false)});
};

function parse_post_body(req, callback) {
	// jesus christ
	var buf = [], len = 0;
	req.on('data', function (data) {
		buf.push(data);
		len += data.length;
	});
	req.once('end', function () {
		var i = 0;
		var dest = new Buffer(len);
		buf.forEach(function (b) {
			b.copy(dest, i, 0);
			i += b.length;
		});
		var combined = dest.toString('utf-8');
		var body = {};
		combined.split('&').forEach(function (param) {
			var m = param.match(/^(.*?)=(.*)$/);
			if (m)
				body[decodeURIComponent(m[1])] = (
					decodeURIComponent(m[2]));
		});
		buf = dest = combined = null;
		callback(null, body);
	});
	req.once('close', callback);
}

function auth_checker(handler, is_post, req, resp, params) {
	if (is_post) {
		parse_post_body(req, function (err, body) {
			if (err) {
				resp.writeHead(500, noCacheHeaders);
				resp.end(preamble + escape(err));
				return;
			}
			req.body = body;
			check_it();
		});
	}
	else
		check_it();

	function check_it() {
		var chunks = twitter.extract_cookie(req.headers.cookie);
		if (!chunks)
			return forbidden('No cookie.');
		twitter.check_cookie(chunks, is_post, ack);
	}

	function ack(err, session) {
		if (err)
			return forbidden(err);
		req.auth = session;
		handler(req, resp, params);
	}

	function forbidden(err) {
		resp.writeHead(401, noCacheHeaders);
		resp.end(preamble + escape(err));
	}
}

exports.route_post = function (pattern, handler) {
	routes.push({method: 'post', pattern: pattern,
			handler: auth_passthrough.bind(null, handler)});
};

exports.route_post_auth = function (pattern, handler) {
	routes.push({method: 'post', pattern: pattern,
			handler: auth_checker.bind(null, handler, true)});
};

exports.enable_debug = function () {
	debug_static.enabled = true;
};

function debug_static(req, resp) {
	/* Highly insecure. */
	var url = req.url.replace(/\.\.+/g, '');
	var path = require('path').join(__dirname, '..', 'www', url);
	var s = require('fs').createReadStream(path);
	s.once('error', function (err) {
		if (err.code == 'ENOENT')
			render_404(resp);
		else {
			resp.writeHead(500, noCacheHeaders);
			resp.end(preamble + escape(err.message));
		}
	});
	s.once('open', function () {
		var h = {};
		try {
			var mime = require('connect').utils.mime;
			var ext = require('path').extname(path);
			h['Content-Type'] = mime.type(ext);
		} catch (e) {}
		resp.writeHead(200, h);
		util.pump(s, resp);
	});
	return true;
};

var vanillaHeaders = {'Content-Type': 'text/html; charset=UTF-8'};
var noCacheHeaders = {'Content-Type': 'text/html; charset=UTF-8',
		'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
		'Cache-Control': 'no-cache'};
var preamble = '<!doctype html><meta charset=utf-8>';

exports.vanillaHeaders = vanillaHeaders;
exports.noCacheHeaders = noCacheHeaders;

exports.notFoundHtml = preamble + '<title>404</title>404';

function render_404(resp) {
	resp.writeHead(404, noCacheHeaders);
	resp.end(exports.notFoundHtml);
};
exports.render_404 = render_404;

exports.redirect = function (resp, uri, code) {
	var headers = {Location: uri};
	for (var k in vanillaHeaders)
		headers[k] = vanillaHeaders[k];
	resp.writeHead(code || 303, headers);
	resp.end(preamble + '<title>Redirect</title>'
		+ '<a href="' + encodeURI(uri) + '">Proceed</a>.');
};

exports.dump_server_error = function (resp, err) {
	resp.writeHead(500, noCacheHeaders);
	resp.write(preamble + '<title>Server error</title>\n<pre>');
	resp.write(escape(util.inspect(err)));
	resp.end('</pre>');
};
