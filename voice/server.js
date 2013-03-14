var caps = require('../server/caps'),
    config = require('../config');
    crypto = require('crypto'),
    db = require('../db'),
    fs = require('fs'),
    imager = require('../imager'),
    joinPath = require('path').join,
    Muggle = require('../muggle').Muggle,
    request = require('request'),
    winston = require('winston'),
    web = require('../server/web');
 
function tts(msg, dest, cb) {
	msg = msg.slice(0, 100);
	var query = {
		q: msg,
		tl: 'ja',
		total: 1,
		idx: 0,
		textlen: msg.length,
		ie: 'UTF-8',
	};
	var headers = {
		Referer: "http://translate.google.com/",
		'User-Agent': "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.22 (KHTML, like Gecko) Chrome/25.0.1364.160 Safari/537.22",
	};
	var packet = {qs: query, encoding: null, headers: headers};
 
	request('http://translate.google.com/translate_tts', packet,
			function (err, resp, body) {
		if (err)
			return cb(err);
		if (resp.statusCode != 200)
			return cb(resp.statusCode || 500);
		fs.writeFile(dest, body, cb);
	});
}

web.resource(/^\/(\w+)\/(\d+)\/voice$/, function (req, params, cb) {
	var board = params[1], num = parseInt(params[2], 10);
	if (!num || !caps.can_access_board(req.ident, board))
		return cb(404);
	var op = db.OPs[num];
	if (!op || !db.OP_has_tag(board, op))
		return cb(404);
	var yaku = new db.Yakusoku(null, req.ident);
	yaku.get_current_body(num, function (err, body) {
		if (err)
			return cb(err);
		body = body && body.trim();
		if (!body)
			return cb(Muggle("No text."));

		var MD5 = crypto.createHash('md5').update(body).digest('hex');
		var MP3 = 'v' + imager.squish_MD5(MD5) + '.mp3';
		var path = joinPath(config.VOICE_PATH, MP3);
		fs.exists(path, function (exists) {
			if (exists)
				return cb(null, 'ok', {path: path});
			tts(body, path, function (err) {
				if (err)
					return cb(err);
				cb(null, 'ok', {path: path});
			});
		});
	});
},
function (req, resp) {
	resp.writeHead(200, {
		'Content-Type': 'audio/mpeg',
		'Cache-Control': 'max-age=600000',
	});
	var stream = fs.createReadStream(this.path);
	stream.pipe(resp);
	stream.once('error', function (err) {
		winston.error(err);
		resp.end();
	});
});
