var _ = require('../lib/underscore'),
    db = require('../db'),
    expat = require('node-expat'),
    request = require('request'),
    winston = require('winston');

var RADIO_IDENT = {auth: 'Radio', ip: '127.0.0.1'};
var RADIO_MOUNT = '/radio';
var POLL_URL = 'http://localhost:5555/poll.xsl';
var M3U_URL = 'http://doushio.com/radio.m3u';
var SHORT_INTERVAL = 3 * 1000;
var LONG_INTERVAL = 30 * 1000;

function update_banner(info, cb) {
	var yaku = new db.Yakusoku(info.board, RADIO_IDENT);
	yaku.set_banner(info.op, info.message, function (err, res) {
		yaku.disconnect();
		cb(err, res);
	});
}

function monitor(last) {
	poll(POLL_URL, function (err, mounts) {
		var info, clear = false;
		if (err)
			winston.error(err);
		else
			info = format_now_playing(mounts);
		var interval = SHORT_INTERVAL;
		if (!info && last) {
			clear = true;
			info = last;
			info.message = '';
			interval = LONG_INTERVAL;
		}
		var sameAsLast = _.isEqual(info, last);
		if (!clear && (!info || sameAsLast)) {
			if (!info)
				interval = LONG_INTERVAL;
			setTimeout(monitor.bind(null, last), interval);
		}
		else {
			update_banner(info, function (err, cb) {
				if (err) {
					winston.error(err);
					interval = LONG_INTERVAL;
				}
				if (clear)
					info = null;
				setTimeout(monitor.bind(null, info), interval);
			});
		}
	});
}

function poll(url, cb) {
	request.get(url, function (err, resp, body) {
		if (err)
			return cb(err);
		if (resp.statusCode != 200)
			return winston.error("Got " + resp.statusCode);
		parse(body, function (err, mounts) {
			if (err)
				cb(err);
			else
				cb(null, mounts);
		});
	});
}

function format_now_playing(mounts) {
	var radio = mounts[RADIO_MOUNT];
	if (!radio || !radio.url)
		return null;
	var m = radio.url.match(/\/(\w+)\/(\d+)/);
	if (!m)
		return null;
	var board = m[1];
	if (!db.is_board(board))
		return;
	var op = parseInt(m[2], 10);
	if (!op)
		return;
	var count = parseInt(radio.listeners, 10);
	count = count + ' listener' + (count == 1 ? '' : 's');
	var msg = [{text: count, href: M3U_URL}];
	if (radio.title)
		msg.push(': ' + radio.title);
	return {board: board, op: op, message: msg};
}

function parse(input, cb) {
	var mounts = {}, curMount, thisKey, thisVal;
	var parser = new expat.Parser('UTF-8');
	parser.on('startElement', function (name, data) {
		if (name == 'mount') {
			curMount = {};
			return;
		}
		if (curMount) {
			thisKey = name;
			thisVal = [];
		}
	});
	parser.on('text', function (val) {
		if (thisKey)
			thisVal.push(val);
	});
	parser.on('endElement', function (name) {
		if (thisKey) {
			if (name != thisKey)
				winston.error("Unmatched or nested?!");
			curMount[thisKey] = thisVal.join('').trim();
			thisKey = thisVal = null;
		}
		else if (name == 'mount') {
			if (curMount.point)
				mounts[curMount.point] = curMount;
			else
				winston.warn("Unknown mount", curMount);
			curMount = null;
		}
		else if (name == 'mounts') {
			parser.removeAllListeners();
			cb(null, mounts);
		}
	});
	parser.on('error', cb);
	parser.parse(input);
}

if (require.main === module) {
	var args = process.argv;
	if (args.length == 2)
		monitor();
	else if (args.length == 5) {
		var op = parseInt(args[3], 10);
		var info = {board: args[2], op: op, message: args[4]};
		update_banner(info, function (err) {
			if (err)
				winston.error(err);
			process.exit(err ? -1 : 0);
		});
	}
}
