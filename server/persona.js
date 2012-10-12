var _ = require('../lib/underscore'),
    config = require('../config'),
    crypto = require('crypto'),
    formidable = require('formidable'),
    https = require('https'),
    querystring = require('querystring'),
    RES = require('./state').resources,
    winston = require('winston');

function connect() {
	if (!RES.sharedConnection)
		RES.sharedConnection = require('../db').redis_client();
	return RES.sharedConnection;
}

exports.login = function (req, resp) {
	try {
		var form = new formidable.IncomingForm();
		form.maxFieldsSize = 50 * 1024;
		form.type = 'urlencoded';
		form.parse(req, verify_persona.bind(null, resp));
	}
	catch (e) {
		winston.error('formidable threw', e);
		respond_error(resp, "Bad request.");
	}
};

function verify_persona(resp, err, fields) {
	if (err) {
		winston.error("Bad POST", err);
		return respond_error(resp, 'POST error.');
	}
	if (!fields.assertion || typeof fields.assertion != 'string')
		return respond_error(resp, 'Bad Persona assertion.');
	var payload = new Buffer(querystring.stringify({
		assertion: fields.assertion,
		audience: config.PERSONA_AUDIENCE,
	}), 'utf8');
	var opts = {
		host: 'verifier.login.persona.org',
		method: 'POST',
		path: '/verify',
		headers: {
			'Content-Length': payload.length,
			'Content-Type': 'application/x-www-form-urlencoded',
		},
	};
	var req = https.request(opts, function (verResp) {
		if (verResp.statusCode != 200) {
			winston.error('Code', verResp.statusCode);
			return respond_error(resp, 'Persona.org error.');
		}
		verResp.once('error', function (err) {
			winston.error("Persona response error", err);
			respond_error(resp, "Couldn't read Persona.");
		});
		verResp.setEncoding('utf8');
		var answer = [];
		verResp.on('data', function (s) {
			answer.push(s);
		});
		verResp.once('end', function () {
			var packet = answer.join('');
			try {
				packet = JSON.parse(packet);
			}
			catch (e) {
				winston.error('Bad packet:', packet);
				return respond_error(resp,
						'Received corrupt Persona.');
			}
			verify_auth(resp, packet);
		});
	});
	req.once('error', function (err) {
		winston.error("Bad persona request", err);
		respond_error(resp, "Couldn't contact persona.org.");
	});
	req.end(payload);
}

function verify_auth(resp, packet) {
	if (!packet || packet.status != 'okay')
		return respond_error(resp, 'Bad Persona.');
	delete packet.status;
	if (packet.audience != config.PERSONA_AUDIENCE) {
		winston.error("Wrong audience: " + packet.audience);
		return respond_error(resp, 'Bad Persona audience.');
	}
	delete packet.audience;
	if (packet.expires && packet.expires < new Date().getTime())
		return respond_error(resp, 'Login attempt expired.');
	var email = packet.email;
	var admin = config.ADMIN_PERSONAS.indexOf(email) >= 0;
	var mod = config.MODERATOR_PERSONAS.indexOf(email) >= 0;
	if (!(admin || mod)) {
		winston.error("Login attempt by " + email);
		return respond_error(resp, 'Wrong Persona.');
	}
	if (admin)
		packet.auth = 'Admin';
	else if (mod)
		packet.auth = 'Moderator';
	else
		delete packet.auth;
	exports.set_cookie(resp, packet);
};

exports.set_cookie = function (resp, info) {
	var pass = random_str();
	info.csrf = random_str();

	var m = connect().multi();
	m.hmset('session:'+pass, info);
	m.expire('session:'+pass, config.LOGIN_SESSION_TIME);
	m.exec(function (err) {
		if (err)
			return oauth_error(resp, err);
		respond_ok(resp, make_cookie('a', pass, info.expires));
	});
};

function extract_login_cookie(chunks) {
	if (!chunks || !chunks.a)
		return false;
	return chunks.a.match(/^[a-zA-Z0-9+\/]{20}$/) ? chunks.a : false;
}
exports.extract_login_cookie = extract_login_cookie;

exports.check_cookie = function (cookie, callback) {
	var r = connect();
	r.hgetall('session:' + cookie, function (err, session) {
		if (err)
			return callback(err);
		else if (_.isEmpty(session))
			return callback('Not logged in.');
		callback(null, session);
	});
};

exports.logout = function (req, resp) {
	var r = connect();
	var chunks = require('./web').parse_cookie(req.headers.cookie);
	var cookie = extract_login_cookie(chunks);
	if (!cookie)
		return respond_error(resp, "No login cookie for logout.");
	r.hgetall('session:' + cookie, function (err, session) {
		if (err)
			return respond_error(resp, "Logout error.");
		r.del('session:' + chunks.a);
		respond_ok(resp, 'a=; expires=Thu, 01 Jan 1970 00:00:00 GMT');
	});
};

function respond_error(resp, message) {
	resp.writeHead(200, {'Content-Type': 'application/json'});
	resp.end(JSON.stringify({status: 'error', message: message}));
}

function respond_ok(resp, cookie) {
	var headers = {
		'Content-Type': 'application/json',
		'Set-Cookie': cookie,
	};
	resp.writeHead(200, headers);
	resp.end(JSON.stringify({status: 'okay'}));
}

function make_expiry() {
	var expiry = new Date(new Date().getTime()
		+ config.LOGIN_SESSION_TIME*1000).toUTCString();
	/* Change it to the expected dash-separated format */
	var m = expiry.match(/^(\w+,\s+\d+)\s+(\w+)\s+(\d+\s+[\d:]+\s+\w+)$/);
	return m ? m[1] + '-' + m[2] + '-' + m[3] : expiry;
}

function make_cookie(key, val) {
	var header = key + '=' + val + '; Expires=' + make_expiry();
	var domain = config.LOGIN_COOKIE_DOMAIN;
	if (domain)
		header += '; Domain=' + domain;
	return header;
}

function random_str() {
	return crypto.randomBytes(15).toString('base64');
}
