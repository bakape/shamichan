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
		/* Generate login info */
		var pass = random_str();
		var second = random_str();
		var info = {csrf: second, token: token, secret: secret,
				user: user, twitter_id: results.user_id};
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
	}
};

function make_expiry() {
	var expiry = new Date(new Date().getTime()
			+ config.LOGIN_SESSION_TIME).toUTCString();
}

function make_cookie(key, val, expiry) {
	var domain = config.LOGIN_COOKIE_DOMAIN;
	return key+'='+val+'; Domain='+domain+'; Path=/; Expires='+expiry;
}

function random_str() {
	return new Buffer((''+Math.random()).substr(2)).toString('base64');
}
