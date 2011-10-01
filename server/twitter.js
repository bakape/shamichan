var config = require('./config'),
    db = require('./db'),
    OAuth = require('oauth').OAuth;

var oauth = new OAuth('https://api.twitter.com/oauth/request_token',
		'https://api.twitter.com/oauth/access_token',
		config.TWITTER_API.key, config.TWITTER_API.secret,
		'1.0', config.TWITTER_API.callback, 'HMAC-SHA1');

function oauth_error(resp, err) {
	if (err)
		console.error(err);
	resp.writeHead(500);
	resp.end('Twitter auth error. Try again later.');
}

exports.login = function (req, resp) {
	oauth.getOAuthRequestToken(function (err, token, secret) {
		if (err)
			return oauth_error(resp, err);
		var uri = 'https://api.twitter.com/oauth/authorize' +
				'?oauth_token=' + encodeURI(token);
		var r = db.redis_client();
		var m = r.multi();
		m.set('oauth:' + token, secret);
		m.expire('oauth:' + token, 5*60);
		m.exec(function (err) {
			r.quit();
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
	var r = db.redis_client();
	var token = req.query.oauth_token;
	r.get('oauth:' + token, function (err, secret) {
		if (err || !secret) {
			r.quit();
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
			r.quit();
			return;
		}
		var user = results.screen_name;
		if (config.TWITTER_USERNAMES.indexOf(user) < 0) {
			resp.writeHead(401);
			resp.end('Invalid user.');
			r.quit();
			return;
		}
		results.token = token;
		results.secret = secret;
		results.user = results.screen_name;
		results.twitter_id = results.user_id;
		delete results.screen_name;
		delete results.user_id;
		exports.set_cookie(resp, results, r);
	}
};

exports.set_cookie = function (resp, info, r) {
	if (!info)
		info = {};
	if (!r)
		r = db.redis_client();
	var pass = random_str();
	var second = random_str();
	info.csrf = second;
	var m = r.multi();
	m.hmset('session:'+pass, info);
	m.expire('session:'+pass, config.LOGIN_SESSION_TIME);
	m.exec(function (err) {
		r.quit();
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

function parse_cookie(header) {
	var chunks = {};
	(header || '').split(';').forEach(function (part) {
		var bits = part.match(/^([^=]*)=(.*)$/);
		if (bits)
			chunks[bits[1].trim()] = bits[2].trim();
	});
	return chunks;
}

exports.check_cookie = function (req, callback) {
	var chunks = parse_cookie(req.headers.cookie);
	var pass = chunks.a;
	if (!pass)
		return callback('Not logged in.');

	var r = db.redis_client();
	r.hgetall('session:' + pass, function (err, session) {
		r.quit();
		if (err)
			callback(err);
		else if (!session || !Object.keys(session).length)
			callback('Not logged in.');
		else
			callback(null, session);
	});
};

function make_expiry() {
	var expiry = new Date(new Date().getTime()
			+ config.LOGIN_SESSION_TIME).toUTCString();
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
