var caps = require('../server/caps'),
    config = require('../config'),
    common = require('../common'),
    db = require('../db'),
    msgcheck = require('../server/msgcheck'),
    okyaku = require('../server/okyaku'),
    recaptcha = require('recaptcha');

const ERRORS = {
	'invalid-site-private-key': "Sorry, the server isn't set up with reCAPTCHA properly.",
	'invalid-request-cookie': "Something went wrong with our reCAPTCHA token. Please try again.",
	'incorrect-captcha-sol': "Incorrect.",
	'captcha-timeout': "Sorry, you took too long. Please try again.",
};

function report(num, cb) {
	// stub
	cb(null);
}

okyaku.dispatcher[common.REPORT_POST] = function (msg, client) {
	if (!msgcheck.check(['id', 'string', 'string'], msg))
		return false;

	var num = msg[0];
	var op = db.OPs[num];
	if (!op || !caps.can_access_thread(client.ident, op))
		return reply_error("Post does not exist.");

	var data = {
		remoteip: client.ident.ip,
		challenge: msg[1],
		response: msg[2].trim(),
	};
	if (!data.challenge || !data.response)
		return reply_error("Pretty please?");
	if (data.challenge.length > 10000 || data.response.length > 10000)
		return reply_error("tl;dr");

	var checker = new recaptcha.Recaptcha(config.RECAPTCHA_PUBLIC_KEY,
			config.RECAPTCHA_PRIVATE_KEY, data);
	checker.verify(function (ok, err) {
		if (!ok) {
			reply_error(ERRORS[err] || err);
			return;
		}

		var op = db.OPs[num];
		if (!op)
			return reply_error("Post does not exist.");
		report(num, function (err) {
			if (err)
				return reply_error(err);
			// success!
			client.send([op, common.REPORT_POST, num]);
		});
	});
	return true;

	function reply_error(err) {
		if (!err)
			err = 'Unknown reCAPTCHA error.';
		var op = db.OPs[num] || 0;
		var msg = {status: 'error', error: err};
		client.send([op, common.REPORT_POST, num, msg]);
		return true;
	}
};

