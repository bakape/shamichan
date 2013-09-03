var caps = require('../server/caps'),
    config = require('../config'),
    common = require('../common'),
    db = require('../db'),
    msgcheck = require('../server/msgcheck'),
    okyaku = require('../server/okyaku'),
    recaptcha = require('recaptcha'),
    winston = require('winston');

const MAIL_FROM = "Reports <reports@doushio.com>";
const URL_BASE = 'http://localhost:8000/';

var SMTP = require('nodemailer').createTransport('SMTP', {
	service: 'Gmail',
	auth: {
		user: "reports@doushio.com",
		pass: "",
	},
});

const ERRORS = {
	'invalid-site-private-key': "Sorry, the server isn't set up with reCAPTCHA properly.",
	'invalid-request-cookie': "Something went wrong with our reCAPTCHA token. Please try again.",
	'incorrect-captcha-sol': "Incorrect.",
	'captcha-timeout': "Sorry, you took too long. Please try again.",
};


function report(reporter_ident, op, num, cb) {

	var board = caps.can_access_thread(reporter_ident, op);
	if (!board)
		return cb("Post does not exist.");

	var noun;
	var url = URL_BASE + board + '/' + op;
	if (op == num) {
		noun = 'Thread';
	}
	else {
		noun = 'Post';
		url += '#' + num;
	}

	var body = url;
	var reporter = reporter_ident.ip;

	var opts = {
		from: MAIL_FROM,
		to: config.REPORT_EMAILS.join(', '),
		subject: noun + ' #' + num + ' reported by ' + reporter,
		text: body,
	};
	SMTP.sendMail(opts, function (err, resp) {
		if (err)
			return cb(err);
		cb(null);
	});
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
		report(client.ident, op, num, function (err) {
			if (err) {
				winston.error(err);
				return reply_error("Couldn't send report.");
			}
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

