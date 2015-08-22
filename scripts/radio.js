/* Polls an icecast2 server for now-playing info, and broadcasts it.
 * Configure here, and install poll.xsl into icecast http root.
 */

var _ = require('underscore'),
    caps = require('../server/caps'),
    db = require('../db'),
    expat = require('node-expat'),
    request = require('request'),
    winston = require('winston');

var RADIO_IDENT = {auth: 'Radio', ip: '127.0.0.1'};
var RADIO_MOUNT = '/radio';
var ICECAST_POLL_URL = 'http://localhost:5555/poll.xsl';
var M3U_URL = 'http://doushio.com/radio.m3u';
var SHORT_INTERVAL = 3 * 1000;
var LONG_INTERVAL = 30 * 1000;

function update_banner(info, cb) {
	var yaku = new db.Yakusoku(info.board, RADIO_IDENT);
	yaku.set_banner(info.msg, function (err, res) {
		yaku.disconnect();
		cb(err, res);
	});
}

function make_monitor(poll) {
function monitor(last) {
	poll(function (err, info) {
		if (err)
			winston.error(err); // fall through
		var clear = false;
		var interval = SHORT_INTERVAL;
		if (!info && last) {
			clear = true;
			info = last;
			info.msg = '';
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
	return monitor;
}

function poll_icecast(cb) {
	request.get(ICECAST_POLL_URL, function (err, resp, body) {
		if (err)
			return cb(err);
		if (resp.statusCode != 200)
			return cb("Got " + resp.statusCode);
		parse_icecast(body, function (err, mounts) {
			if (err)
				cb(err);
			else
				cb(null, format_icecast(mounts));
		});
	});
}

function format_icecast(mounts) {
	var radio = mounts[RADIO_MOUNT];
	if (!radio || !radio.url)
		return null;
	var info = extract_thread(radio.url);
	if (!info)
		return null;
	var count = parseInt(radio.listeners, 10);
	count = count + ' listener' + (count == 1 ? '' : 's');
	var msg = [{text: count, href: M3U_URL}];
	if (radio.title)
		msg.push(': ' + radio.title);
	info.msg = msg;
	return info;
}

function extract_thread(url) {
	var m = /\/(\w+)\/(\d+)/.exec(url);
	if (!m)
		return;
	var board = m[1];
	if (!db.is_board(board) || !caps.can_access_board(RADIO_IDENT, board))
		return;
	var op = parseInt(m[2], 10);
	if (!op)
		return;
	return {board: board, op: op};
}

function parse_icecast(input, cb) {
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
				winston.warn("Unknown mount: " + curMount);
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

var R_A_D_IO_POLL_URL = 'http://r-a-d.io/api';

function poll_r_a_d_io(cb) {
	var opts = {
		url: R_A_D_IO_POLL_URL,
		json: true,
	};
	request.get(opts, function (err, resp, body) {
		if (err)
			return cb(err);
		if (resp.statusCode != 200)
			return cb("Got " + resp.statusCode);
		cb(null, format_r_a_d_io(body));
	});
}

function format_r_a_d_io(json) {
	if (!json || !json.main)
		return null;
	var station = json.main;
	var info = extract_thread(station.thread);
	if (!info)
		return null;
	var count = station.listeners || '???';
	count = count + ' listener' + (count == 1 ? '' : 's');
	var msg = [{text: count, href: 'http://r-a-d.io/'}];

	var np = station.np;
	if (typeof np == 'string' && np) {
		msg.push(': ' + np.slice(0, 100));
	}
	info.msg = msg;
	return info;
}

var reduce_regexp = /&(?:amp|lt|gt|quot);/g;
var reductions = {'&amp;' : '&', '&lt;': '<', '&gt;': '>', '&quot;': '"'};
function reduce_entities(html) {
	return html.replace(reduce_regexp, function (c) {
		return reductions[c];
	});
}

if (require.main === module) {
	var args = process.argv;
	if (args.length == 2) {
		winston.info('Polling ' + ICECAST_POLL_URL + '.');
		make_monitor(poll_icecast)();
	}
	else if (args.length == 5) {
		var op = parseInt(args[3], 10);
		var info = {board: args[2], op: op, msg: args[4]};
		update_banner(info, function (err) {
			if (err)
				winston.error(err);
			process.exit(err ? 1 : 0);
		});
	}
	else if (args[2] == '--r-a-d-io') {
		winston.info('Polling ' + R_A_D_IO_POLL_URL + '.');
		make_monitor(poll_r_a_d_io)();
	}
}
