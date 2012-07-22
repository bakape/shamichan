var _ = require('../lib/underscore'),
    caps = require('./caps'),
    config = require('../config'),
    formidable = require('formidable'),
    twitter = require('./twitter'),
    url_parse = require('url').parse,
    util = require('util');

var escape = require('../common').escape_html;
var routes = [];

var server = require('http').createServer(function (req, resp) {
	var ip = req.connection.remoteAddress;
	if (config.TRUST_X_FORWARDED_FOR)
		ip = parse_forwarded_for(req.headers['x-forwarded-for']) || ip;
	if (!ip)
		throw "No IP?!";
	req.ident = caps.lookup_ident(ip);
	if (req.ident.ban)
		return render_500(resp);
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

function parse_forwarded_for(ff) {
	if (!ff)
		return null;
	if (ff.indexOf(',') >= 0)
		ff = ff.split(',', 1)[0];
	return ff.trim();
}
exports.parse_forwarded_for = parse_forwarded_for;

function auth_passthrough(handler, req, resp, params) {
	var chunks = twitter.extract_cookie(req.headers.cookie);
	if (!chunks) {
		handler(req, resp, params);
		return;
	}

	twitter.check_cookie(chunks, false, function (err, ident) {
		if (!err)
			_.extend(req.ident, ident);
		handler(req, resp, params);
	});
}

exports.route_get_auth = function (pattern, handler) {
	routes.push({method: 'get', pattern: pattern,
			handler: auth_checker.bind(null, handler, false)});
};

function auth_checker(handler, is_post, req, resp, params) {
	if (is_post) {
		var form = new formidable.IncomingForm();
		form.maxFieldsSize = 50 * 1024;
		form.type = 'urlencoded';
		form.parse(req, function (err, fields) {
			if (err) {
				resp.writeHead(500, noCacheHeaders);
				resp.end(preamble + escape(err));
				return;
			}
			req.body = fields;
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
		_.extend(req.ident, session);
		handler(req, resp, params);
	}

	function forbidden(err) {
		resp.writeHead(401, noCacheHeaders);
		resp.end(preamble + escape(err));
	}
}

exports.route_post = function (pattern, handler) {
	// auth_passthrough conflicts with formidable
	// (by the time the cookie check comes back, formidable can't
	// catch the form data)
	// We don't need the auth here anyway currently thanks to client_id
	routes.push({method: 'post', pattern: pattern, handler: handler});
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
			h['Content-Type'] = require('mime').lookup(path);
		} catch (e) {}
		resp.writeHead(200, h);
		util.pump(s, resp);
	});
	return true;
};

var vanillaHeaders = {'Content-Type': 'text/html; charset=UTF-8'};
var noCacheHeaders = {'Content-Type': 'text/html; charset=UTF-8',
		'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
		'Cache-Control': 'no-cache, no-store'};
var preamble = '<!doctype html><meta charset=utf-8>';

exports.vanillaHeaders = vanillaHeaders;
exports.noCacheHeaders = noCacheHeaders;

exports.notFoundHtml = preamble + '<title>404</title>404';
exports.serverErrorHtml = preamble + '<title>500</title>Server error';

function render_404(resp) {
	resp.writeHead(404, noCacheHeaders);
	resp.end(exports.notFoundHtml);
};
exports.render_404 = render_404;

function render_500(resp) {
	resp.writeHead(500, noCacheHeaders);
	resp.end(exports.serverErrorHtml);
}
exports.render_500 = render_500;

exports.redirect = function (resp, uri, code) {
	var headers = {Location: uri};
	for (var k in vanillaHeaders)
		headers[k] = vanillaHeaders[k];
	resp.writeHead(code || 303, headers);
	resp.end(preamble + '<title>Redirect</title>'
		+ '<a href="' + encodeURI(uri) + '">Proceed</a>.');
};

var redirectJsTmpl = require('fs').readFileSync('tmpl/redirect.html');

exports.redirect_js = function (resp, uri) {
	resp.writeHead(200, web.noCacheHeaders);
	resp.write(preamble + '<title>Redirecting...</title>');
	resp.write('<script>var dest = "' + encodeURI(uri) + '";</script>');
	resp.end(redirectJsTmpl);
};

exports.dump_server_error = function (resp, err) {
	resp.writeHead(500, noCacheHeaders);
	resp.write(preamble + '<title>Server error</title>\n<pre>');
	resp.write(escape(util.inspect(err)));
	resp.end('</pre>');
};
