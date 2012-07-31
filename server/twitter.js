var _ = require('../lib/underscore'),
    config = require('../config'),
    RES = require('./state').resources,
    OAuth = require('oauth').OAuth,
    winston = require('winston');

var oauth = new OAuth('https://api.twitter.com/oauth/request_token',
		'https://api.twitter.com/oauth/access_token',
		config.TWITTER_API.key, config.TWITTER_API.secret,
		'1.0', config.TWITTER_API.callback, 'HMAC-SHA1');

function connect() {
	if (!RES.sharedConnection)
		RES.sharedConnection = require('../db').redis_client();
	return RES.sharedConnection;
}

function oauth_error(resp, err) {
	if (err)
		winston.error(err);
	resp.writeHead(500);
	resp.end('Twitter auth error. Try again later.');
}

exports.login = function (req, resp) {
	oauth.getOAuthRequestToken(function (err, token, secret) {
		if (err)
			return oauth_error(resp, err);
		var uri = 'https://api.twitter.com/oauth/authorize' +
				'?oauth_token=' + encodeURI(token);
		var r = connect();
		r.setex('oauth:' + token, 5*60, secret, function (err) {
			if (err)
				return oauth_error(resp, err);
			resp.writeHead(307, {Location: uri,
					'Content-Type': 'text/html'});
			resp.end('<!doctype html><a href="' + encodeURI(uri)
					+ '">Proceed to Twitter</a>.');
		});
	});
};

exports.verify = function (req, resp) {
	var r = connect();
	var token = req.query.oauth_token;
	r.get('oauth:' + token, function (err, secret) {
		if (err || !secret) {
			return oauth_error(resp, err);
		}
		r.del('oauth:' + token);
		var verifier = req.query.oauth_verifier;
		oauth.getOAuthAccessToken(token, secret, verifier, go);
	});
	function go(err, token, secret, results) {
		if (err) {
			if (parseInt(err.statusCode) == 401) {
				resp.writeHead(401);
				resp.end('Permission failure.');
			}
			else
				oauth_error(resp, err);
			return;
		}
		var user = results.screen_name;
		var admin = config.TWITTER_ADMINS.indexOf(user) >= 0;
		var mod = config.TWITTER_MODERATORS.indexOf(user) >= 0;
		if (!admin && !mod) {
			resp.writeHead(401);
			resp.end('Invalid user.');
			return;
		}
		results.token = token;
		results.secret = secret;
		results.user = results.screen_name;
		results.twitter_id = results.user_id;
		if (admin)
			results.auth = 'Admin';
		else if (mod)
			results.auth = 'Moderator';
		else
			delete results.auth;
		delete results.screen_name;
		delete results.user_id;
		exports.set_cookie(resp, results, r);
	}
};

exports.set_cookie = function (resp, info, r) {
	if (!r)
		r = connect();
	var pass = random_str();
	var second = random_str();
	info.csrf = second;
	var m = r.multi();
	m.hmset('session:'+pass, info);
	m.expire('session:'+pass, config.LOGIN_SESSION_TIME);
	m.exec(function (err) {
		if (err)
			return oauth_error(resp, err);
		var expiry = make_expiry();
		var cookies = [make_cookie('a', pass, expiry),
				make_cookie('b', second, expiry)];
		var headers = {Location: '.', 'Set-Cookie': cookies};
		resp.writeHead(303, headers);
		resp.end('Logged in!');
	});
};

exports.extract_cookie = function (chunks) {
	return chunks.a ? chunks : false;
};

exports.check_cookie = function (chunks, check_csrf, callback) {
	var r = connect();
	r.hgetall('session:' + chunks.a, function (err, session) {
		if (err)
			return callback(err);
		else if (_.isEmpty(session))
			return callback('Not logged in.');
		if (check_csrf) {
			if (!session.csrf)
				return callback('Corrupt session.');
			if (chunks.b !== session.csrf)
				return callback('Possible CSRF.');
		}
		callback(null, session);
	});
};

exports.logout = function (req, resp) {
	var r = connect();
	var chunks = require('./web').parse_cookie(req.headers.cookie);
	r.hgetall('session:' + chunks.a, function (err, session) {
		if (err)
			return fail(err);
		if (session.csrf && chunks.b !== session.csrf)
			return fail('Possible CSRF: ' + chunks.b);
		if (!_.isEmpty(session)) {
			// not a huge deal if this fails
			r.del('session:' + chunks.a);
		}
		var cookies = 'a=; b=; expires=Thu, 01 Jan 1970 00:00:00 GMT';
		var headers = {Location: '..', 'Set-Cookie': cookies};
		resp.writeHead(303, headers);
		resp.end('Logged out.');
	});
	function fail(err) {
		winston.error(err);
		resp.writeHead(500);
		resp.end('Logout failure.');
	}
};

function make_expiry() {
	var expiry = new Date(new Date().getTime()
			+ config.LOGIN_SESSION_TIME*1000).toUTCString();
	/* Change it to the expected dash-separated format */
	var m = expiry.match(/^(\w+,\s+\d+)\s+(\w+)\s+(\d+\s+[\d:]+\s+\w+)$/);
	return m ? m[1] + '-' + m[2] + '-' + m[3] : expiry;
}

function make_cookie(key, val, expiry) {
	var header = key + '=' + val + '; Expires=' + expiry;
	var domain = config.LOGIN_COOKIE_DOMAIN;
	if (domain)
		header += '; Domain=' + domain;
	return header;
}

function random_str() {
	return new Buffer((''+Math.random()).substr(2)).toString('base64');
}
