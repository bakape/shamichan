var _ = require('../lib/underscore'),
    caps = require('./caps'),
    config = require('../config'),
    formidable = require('formidable'),
    persona = require('./persona'),
    url_parse = require('url').parse,
    util = require('util'),
    winston = require('winston');

var escape = require('../common').escape_html;
var routes = [];
var resources = [];

var server = require('http').createServer(function (req, resp) {
	var ip = req.connection.remoteAddress;
	if (config.TRUST_X_FORWARDED_FOR)
		ip = parse_forwarded_for(req.headers['x-forwarded-for']) || ip;
	if (!ip)
		throw "No IP?!";
	req.ident = caps.lookup_ident(ip);
	if (req.ident.ban)
		return render_500(resp);
	var method = req.method.toLowerCase();
	var parsed = url_parse(req.url, true);
	req.url = parsed.pathname;
	req.query = parsed.query;

	var numRoutes = routes.length;
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

	if (method == 'get' || method == 'head')
		for (var i = 0; i < resources.length; i++)
			if (handle_resource(req, resp, resources[i]))
				return;

	if (debug_static.enabled)
		debug_static(req, resp);
	else
		render_404(resp);
});
exports.server = server;

function handle_resource(req, resp, resource) {
	var m = req.url.match(resource.pattern);
	if (!m)
		return false;
	var args = [req];
	if (resource.headParams)
		args.push(m);
	args.push(resource_second_handler.bind(null, req, resp, resource));

	var chunks = parse_cookie(req.headers.cookie);
	var cookie = persona.extract_login_cookie(chunks);
	if (cookie) {
		persona.check_cookie(cookie, function (err, ident) {
			if (err && !resource.authPassthrough)
				return forbidden(resp, 'No cookie.');
			else if (!err)
				_.extend(req.ident, ident);
			resource.head.apply(null, args);
		});
	}
	else if (!resource.authPassthrough)
		render_404(resp);
	else
		resource.head.apply(null, args);
	return true;
}

function resource_second_handler(req, resp, resource, err, act, arg) {
	var method = req.method.toLowerCase();
	if (err) {
		if (err == 404)
			return render_404(resp);
		else if (err != 500)
			winston.error(err);
		return render_500(resp);
	}
	else if (act == 'ok') {
		if (method == 'head') {
			var headers = (arg && arg.headers) || vanillaHeaders;
			resp.writeHead(200, headers);
			resp.end();
			if (resource.tear_down)
				resource.tear_down.call(arg);
		}
		else {
			if (resource.tear_down)
				arg.finished = function () {
					resource.tear_down.call(arg);
				};
			resource.get.call(arg, req, resp);
		}
	}
	else if (act == 304) {
		resp.writeHead(304);
		resp.end();
	}
	else if (act == 'redirect' || (act >= 300 && act < 400)) {
		var headers = {Location: arg};
		if (act == 'redirect')
			act = 303;
		else if (act == 303.1) {
			act = 303;
			headers['X-Robots-Tag'] = 'nofollow';
		}
		resp.writeHead(act, headers);
		resp.end();
	}
	else if (act == 'redirect_js') {
		if (method == 'head') {
			resp.writeHead(303, {Location: arg});
			resp.end();
		}
		else
			redirect_js(resp, arg);
	}
	else
		throw new Error("Unknown resource handler: " + act);
}

exports.route_get = function (pattern, handler) {
	routes.push({method: 'get', pattern: pattern,
			handler: auth_passthrough.bind(null, handler)});
};

exports.resource = function (pattern, head, get, tear_down) {
	var res = {pattern: pattern, head: head, authPassthrough: true};
	res.headParams = (head.length == 3);
	if (get)
		res.get = get;
	if (tear_down)
		res.tear_down = tear_down;
	resources.push(res);
};

exports.resource_auth = function (pattern, head, get, finished) {
	var res = {pattern: pattern, head: head, authPassthrough: false};
	res.headParams = (head.length == 3);
	if (get)
		res.get = get;
	if (finished)
		res.finished = finished;
	resources.push(res);
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
	var chunks = parse_cookie(req.headers.cookie);
	var cookie = persona.extract_login_cookie(chunks);
	if (!cookie) {
		handler(req, resp, params);
		return;
	}

	persona.check_cookie(cookie, function (err, ident) {
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
		try {
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
		catch (e) {
			winston.error('formidable threw', e);
			return forbidden(resp, 'Bad request.');
		}
	}
	else
		check_it();

	function check_it() {
		var chunks = parse_cookie(req.headers.cookie);
		cookie = persona.extract_login_cookie(chunks);
		if (!cookie)
			return forbidden(resp, 'No cookie.');
		persona.check_cookie(cookie, ack);
	}

	function ack(err, session) {
		if (err)
			return forbidden(resp, err);
		if (is_post && session.csrf != req.body.csrf)
			return forbidden(resp, "Possible CSRF.");
		_.extend(req.ident, session);
		handler(req, resp, params);
	}
}

function forbidden(resp, err) {
	resp.writeHead(401, noCacheHeaders);
	resp.end(preamble + escape(err));
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

function redirect(resp, uri, code) {
	var headers = {Location: uri};
	for (var k in vanillaHeaders)
		headers[k] = vanillaHeaders[k];
	resp.writeHead(code || 303, headers);
	resp.end(preamble + '<title>Redirect</title>'
		+ '<a href="' + encodeURI(uri) + '">Proceed</a>.');
}
exports.redirect = redirect;

var redirectJsTmpl = require('fs').readFileSync('tmpl/redirect.html');

function redirect_js(resp, uri) {
	resp.writeHead(200, noCacheHeaders);
	resp.write(preamble + '<title>Redirecting...</title>');
	resp.write('<script>var dest = "' + encodeURI(uri) + '";</script>');
	resp.end(redirectJsTmpl);
}
exports.redirect_js = redirect_js;

exports.dump_server_error = function (resp, err) {
	resp.writeHead(500, noCacheHeaders);
	resp.write(preamble + '<title>Server error</title>\n<pre>');
	resp.write(escape(util.inspect(err)));
	resp.end('</pre>');
};

function parse_cookie(header) {
	var chunks = {};
	(header || '').split(';').forEach(function (part) {
		var bits = part.match(/^([^=]*)=(.*)$/);
		if (bits)
			chunks[bits[1].trim()] = bits[2].trim();
	});
	return chunks;
}
exports.parse_cookie = parse_cookie;
