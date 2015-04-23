/*
 This file is used both by the server and client
 Keep that in mind, when making modifications
 */

// Runing on the server
var isNode = exports.isNode = typeof navigator === 'undefined';

// Define vars, for the server and client sides
var DEF = exports, state, config, hotConfig, imagerConfig, main;
if (isNode) {
	state = require('../server/state');
	config = require('../config');
	hotConfig = state.hot;
	imagerConfig = require('../imager/config');
}
else {
	main = require('./../alpha/main');
	state = require('./../alpha/state');
	config = state.config.attributes;
	hotConfig = state.hotConfig.attributes;
	imagerConfig = state.imagerConfig.attributes;
}

// Websocket message codes
DEF.INVALID = 0;

DEF.INSERT_POST = 2;
DEF.UPDATE_POST = 3;
DEF.FINISH_POST = 4;
// Legacy?
DEF.CATCH_UP = 5;
DEF.INSERT_IMAGE = 6;
DEF.SPOILER_IMAGES = 7;
DEF.DELETE_IMAGES = 8;
DEF.DELETE_POSTS = 9;
DEF.DELETE_THREAD = 10;
DEF.LOCK_THREAD = 11;
DEF.UNLOCK_THREAD = 12;
DEF.REPORT_POST = 13;

DEF.IMAGE_STATUS = 31;
DEF.SYNCHRONIZE = 32;
DEF.EXECUTE_JS = 33;
DEF.MOVE_THREAD = 34;
DEF.UPDATE_BANNER = 35;
DEF.TEARDOWN = 36;
DEF.ONLINE_COUNT = 37;
DEF.HOT_INJECTION = 38;
DEF.NOTIFICATION = 39;
DEF.RADIO = 40;
DEF.RESYNC = 41;

DEF.MODEL_SET = 50;
DEF.COLLECTION_RESET = 55;
DEF.COLLECTION_ADD = 56;
DEF.SUBSCRIBE = 60;
DEF.UNSUBSCRIBE = 61;
DEF.GET_TIME = 62;

DEF.INPUT_ROOM = 20;
DEF.MAX_POST_LINES = 30;
DEF.MAX_POST_CHARS = 2000;
DEF.WORD_LENGTH_LIMIT = 300;

DEF.S_NORMAL = 0;
DEF.S_BOL = 1;
DEF.S_QUOTE = 2;
DEF.S_SPOIL = 3;

function is_pubsub(t) {
	return t > 0 && t < 30;
}
exports.is_pubsub = is_pubsub;

// Finite State Machine
function FSM(start) {
	this.state = start;
	this.spec = {acts: {}, ons: {}, wilds: {}, preflights: {}};
}
exports.FSM = FSM;

FSM.prototype.clone = function() {
	var second = new FSM(this.state);
	second.spec = this.spec;
	return second;
};

// Handlers on arriving to a new state
FSM.prototype.on = function(key, f) {
	var ons = this.spec.ons[key];
	if (ons)
		ons.push(f);
	else
		this.spec.ons[key] = [f];
	return this;
};

// Sanity checks before attempting a transition
FSM.prototype.preflight = function(key, f) {
	var pres = this.spec.preflights[key];
	if (pres)
		pres.push(f);
	else
		this.spec.preflights[key] = [f];
};

// Specify transitions and an optional handler function
FSM.prototype.act = function(trans_spec, on_func) {
	var halves = trans_spec.split('->');
	if (halves.length != 2)
		throw new Error("Bad FSM spec: " + trans_spec);
	var parts = halves[0].split(',');
	var dest = halves[1].match(/^\s*(\w+)\s*$/)[1];
	var tok;
	for (var i = parts.length - 1; i >= 0; i--) {
		var part = parts[i];
		var m = part.match(/^\s*(\*|\w+)\s*(?:\+\s*(\w+)\s*)?$/);
		if (!m)
			throw new Error("Bad FSM spec portion: " + part);
		if (m[2])
			tok = m[2];
		if (!tok)
			throw new Error("Tokenless FSM action: " + part);
		var src = m[1];
		if (src == '*')
			this.spec.wilds[tok] = dest;
		else {
			var acts = this.spec.acts[src];
			if (!acts)
				this.spec.acts[src] = acts = {};
			acts[tok] = dest;
		}
	}
	if (on_func)
		this.on(dest, on_func);
	return this;
};

FSM.prototype.feed = function(ev, param) {
	var spec = this.spec;
	var from = this.state, acts = spec.acts[from];
	var to = (acts && acts[ev]) || spec.wilds[ev];
	if (to && from != to) {
		var ps = spec.preflights[to];
		for (var i = 0; ps && i < ps.length; i++) {
			if (!ps[i].call(this, param))
				return false;
		}
		this.state = to;
		var fs = spec.ons[to];
		for (i = 0; fs && i < fs.length; i++)
			fs[i].call(this, param);
	}
	return true;
};

FSM.prototype.feeder = function(ev) {
	var self = this;
	return function(param) {
		self.feed(ev, param);
	};
};

var entities = {'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;'};
function escape_html(html) {
	return html.replace(/[&<>"]/g, function(c) {
		return entities[c];
	});
}
exports.escape_html = escape_html;

function escape_fragment(frag) {
	var t = typeof (frag);
	if (t == 'object' && frag && typeof (frag.safe) == 'string')
		return frag.safe;
	else if (t == 'string')
		return escape_html(frag);
	else if (t == 'number')
		return frag.toString();
	else
		return '???';
}
exports.escape_fragment = escape_fragment;

function flatten(frags) {
	var out = [];
	for (var i = 0; i < frags.length; i++) {
		var frag = frags[i];
		if (Array.isArray(frag))
			out = out.concat(flatten(frag));
		else
			out.push(escape_fragment(frag));
	}
	return out;
}
exports.flatten = flatten;

function safe(frag) {
	return {safe: frag};
}
exports.safe = safe;

function is_noko(email) {
	return email && email.indexOf('@') == -1 && /noko/i.test(email);
}
exports.is_noko = is_noko;

function is_sage(email) {
	return hotConfig.SAGE_ENABLED && email &&
		email.indexOf('@') == -1 && /sage/i.test(email);
}
exports.is_sage = is_sage;

// TODO: Move to admin.js, when I get to it
function override(obj, orig, upgrade) {
	var origFunc = obj[orig];
	obj[orig] = function() {
		var args = [].slice.apply(arguments);
		args.unshift(origFunc);
		return upgrade.apply(this, args);
	};
}

// Construct hash command regex pattern
var dice_re = '(#flip|#8ball|#sw(?:\\d{1,2}:)?\\d{1,2}:\\d{1,2}(?:[+-]\\d+)?' +
	'|#\\d{0,2}d\\d{1,4}(?:[+-]\\d{1,4})?';
if (config.PYU)
	dice_re += '|#pyu|#pcount';
if (config.RADIO)
	dice_re += '|#q';
dice_re += ')';
dice_re = new RegExp(dice_re, 'i');
exports.dice_re = dice_re;

function parse_dice(frag) {
	if (frag == '#flip')
		return {n: 1, faces: 2};
	if (frag == '#8ball')
		return {n: 1, faces: hotConfig.EIGHT_BALL.length};
	// Increment counter
	if (frag == '#pyu')
		return {pyu: 'increment'};
	// Print current count
	if (frag == '#pcount')
		return {pyu: 'print'};
	if (frag == '#q')
		return {q: true};
	var m = frag.match(/^#(\d*)d(\d+)([+-]\d+)?$/i);
	// Regular dice
	if (m) {
		var n = parseInt(m[1], 10) || 1, faces = parseInt(m[2], 10);
		if (n < 1 || n > 10 || faces < 2 || faces > 100)
			return false;
		var info = {n: n, faces: faces};
		if (m[3])
			info.bias = parseInt(m[3], 10);
		return info;
	}
	// First capture group may or may not be present
	var sw = frag.match(/^#sw(\d+:)?(\d+):(\d+)([+-]\d+)?$/i);
	if (sw) {
		var hour = parseInt(sw[1], 10) || 0,
			min = parseInt(sw[2], 10),
			sec = parseInt(sw[3], 10);
		var time = serverTime();
		// Offset the start. If the start is in the future,
		// a countdown will be displayed
		if (sw[4]) {
			var symbol = sw[4].slice(0, 1);
			var offset = sw[4].slice(1) * 1000;
			time = symbol == '+' ? time + offset : time - offset;
		}
		var end = ((hour * 60 + min) * 60 + sec) * 1000 + time;
		return {hour: hour, min: min, sec: sec, start: time, end: end};
	}
}
exports.parse_dice = parse_dice;

var serverTime = exports.serverTime = function() {
	var d = new Date().getTime();
	// On the server or time difference not compared yet
	if (isNode || !main.serverTimeOffset)
		return d;
	return d + main.serverTimeOffset;
};

var readable_dice = exports.readable_dice = function(bit, d) {
	if (bit == '#flip')
		return '#flip (' + (d[1] == 2) + ')';
	if (bit == '#8ball')
		return '#8ball (' + hotConfig.EIGHT_BALL[d[1] - 1] + ')';
	if (bit == '#pyu')
		return '#pyu(' + d + ')';
	if (bit == '#pcount')
		return '#pcount(' + d + ')';
	if (bit == '#q')
		return '#q (' + d[0] + ')';
	if (/^#sw/.test(bit)) {
		return safe('<syncwatch class="embed" start=' + d[0].start +
			" end=" + d[0].end +
			" hour=" + d[0].hour +
			" min=" + d[0].min +
			" sec=" + d[0].sec +
			' >syncwatch</syncwatch>');
	}
	var n = d.length, b = 0;
	if (d[n - 1] && typeof d[n - 1] == 'object') {
		b = d[n - 1].bias;
		n--;
	}
	var r = d.slice(1, n);
	n = r.length;
	bit += ' (';
	var eq = n > 1 || b;
	if (eq)
		bit += r.join(', ');
	if (b)
		bit += (b < 0 ? ' - ' + (-b) : ' + ' + b);
	var sum = b;
	for (var j = 0; j < n; j++)
		sum += r[j];
	return bit + (eq ? ' = ' : '') + sum + ')';
};

var pick_spoiler = exports.pick_spoiler = function(metaIndex) {
	var imgs = imagerConfig.SPOILER_IMAGES;
	var n = imgs.length;
	var i;
	if (metaIndex < 0)
		i = Math.floor(Math.random() * n);
	else
		i = metaIndex % n;
	return {index: imgs[i], next: (i + 1) % n};
};

var new_tab_link = exports.new_tab_link
	= function(srcEncoded, inside, cls, brackets) {
	if (brackets)
		inside = '[' + inside + '] ';
	return [
		safe('<a href="' + srcEncoded + '" target="_blank"' +
			(cls ? ' class="' + cls + '"' : '') +
			' rel="nofollow">'), inside, safe('</a>')
	];
};

exports.thumbStyles = ['small', 'sharp', 'hide'];

var readable_filesize = exports.readable_filesize = function(size) {
	/* Dealt with it. */
	if (size < 1024)
		return size + ' B';
	if (size < 1048576)
		return Math.round(size / 1024) + ' KB';
	size = Math.round(size / 104857.6).toString();
	return size.slice(0, -1) + '.' + size.slice(-1) + ' MB';
};

var pad = exports.pad = function(n) {
	return (n < 10 ? '0' : '') + n;
};

// Various UI-related links wrapped in []
var action_link_html = exports.action_link_html
	= function(href, name, id, cls) {
	return '<span class="act"><a href="' + href + '"'
		+ (id ? ' id="' + id + '"' : '')
		+ (cls ? ' class="' + cls + '"' : '')
		+ '>' + name + '</a></span>';
};

var reasonable_last_n = exports.reasonable_last_n = function(n) {
	return Number.isInteger(n) && n >= 5 && n <= 500;
};

var parse_name = exports.parse_name = function(name) {
	var tripcode = '', secure = '';
	var hash = name.indexOf('#');
	if (hash >= 0) {
		tripcode = name.substr(hash + 1);
		name = name.substr(0, hash);
		hash = tripcode.indexOf('#');
		if (hash >= 0) {
			secure = escape_html(tripcode.substr(hash + 1));
			tripcode = tripcode.substr(0, hash);
		}
		tripcode = escape_html(tripcode);
	}
	name = name.trim().replace(hotConfig.EXCLUDE_REGEXP, '');
	return [
		name.substr(0, 100), tripcode.substr(0, 128),
		secure.substr(0, 128)
	];
};

exports.random_id = function() {
	return Math.floor(Math.random() * 1e16) + 1;
};

// Required at the end to avoid circular dependancy incompleteness
exports.OneeSama = require('./oneesama');
