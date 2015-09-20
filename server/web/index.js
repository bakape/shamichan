/*
 Webserver
 */

const _ = require('underscore'),
	admin = require('./admin'),
	api = require('./api'),
	caps = require('../caps'),
	cookieParser = require('cookie-parser'),
	compress = require('compression'),
	config = require('../../config'),
	express = require('express'),
	html = require('./html'),
	http = require('http'),
	imager = require('../../imager/daemon'),
	persona = require('../persona'),
	util = require('./util'),
	websocket = require('./websocket');

const app = express(),
	server = http.createServer(app);

app.enable('strict routing').disable('etag');
server.listen(config.LISTEN_PORT);

// NOTE: Order is important as it determines handler priority

app.use(cookieParser());

// Pass the client IP through authentication checks
app.use(function(req, res, next) {
	let ip = req.connection.remoteAddress;
	if (config.TRUST_X_FORWARDED_FOR)
		ip = util.parse_forwarded_for(req.headers['x-forwarded-for']) || ip;
	if (!ip) {
		res.set({'Content-Type': 'text/plain'});
		res.status(500).send(
			"Your IP could not be determined. This server is misconfigured."
		);
		return;
	}
	req.ident = caps.lookup_ident(ip);
	// TODO: A prettier ban page would be nice, once we have actual ban comments
	if (req.ident.ban)
		return util.send404(res);

	// Staff authentication
	const loginCookie = persona.extract_login_cookie(req.cookies);
	if (loginCookie) {
		persona.check_cookie(loginCookie, function (err, ident) {
			if (!err)
				_.extend(req.ident, ident);
			next();
		})
	}
	else
		next();
});

websocket.start(server);
if (config.GZIP)
	app.use(compress());
app.post('/login', persona.login)
	.post('/logout', persona.logout)
	.post('/upload/', imager.new_upload)
	.use(admin)
	.use('/api/', api);
if (config.SERVE_STATIC_FILES) {
	const opts = {};
	if (!config.DEBUG) {
		opts.etag = false;
		opts.maxAge = '350 days';
	}
	else
		opts.setHeaders = res => res.set(util.noCacheHeaders);
	app.use(express.static('www', opts));
}

app.use(html);

// No match on other routers
app.use((req, res) => util.send404(res));
