// This file is used both by the server and client
// Keep that in mind, when making modifications

// Runing on the server
var isNode = typeof navigator === 'undefined';

// Define vars, for the server and client sides
var DEF = exports, state, config, hotConfig, imagerConfig;
if (isNode) {
	state = require('./server/state');
	config = require('./config');
	hotConfig = state.hot;
	imagerConfig = require('./imager/config');
}
else {
	state = require('./alpha/state');
	config = state.config.attributes;
	hotConfig = state.hotConfig.attributes;
	imagerConfig = state.imagerConfig.attributes;
}

var lang = require('./lang/');

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

if (typeof mediaURL == 'undefined' || !mediaURL)
	mediaURL = imagerConfig.MEDIA_URL;

function is_pubsub(t) {
	return t > 0 && t < 30;
}
exports.is_pubsub = is_pubsub;

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
		for (var i = 0; ps && i < ps.length; i++)
			if (!ps[i].call(this, param))
				return false;
		this.state = to;
		var fs = spec.ons[to];
		for (var i = 0; fs && i < fs.length; i++)
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

var OneeSama = function(t) {
	this.tamashii = t;
	this.hooks = {};
};
exports.OneeSama = OneeSama;

var OS = OneeSama.prototype;

var break_re = new RegExp("(\\S{" + DEF.WORD_LENGTH_LIMIT + "})");

// Internal refs, embeds
var ref_re = '>>(\\d+';
ref_re += '|>\\/watch\\?v=[\\w-]{11}(?:#t=[\\dhms]{1,9})?';
ref_re += '|>\\/soundcloud\\/[\\w-]{1,40}\\/[\\w-]{1,80}';
ref_re += '|>\\/pastebin\\/\\w+';

for (var i = 0; i < config.BOARDS.length; i++) {
	ref_re += '|>\\/' + config.BOARDS[i] + '\\/(?:\\d+)?';
}

ref_re += ')';
ref_re = new RegExp(ref_re);

OS.hook = function(name, func) {
	var hs = this.hooks[name];
	if (!hs)
		this.hooks[name] = hs = [func];
	else if (hs.indexOf(func) < 0)
		hs.push(func);
};

OS.trigger = function(name, param) {
	var hs = this.hooks[name];
	if (hs)
		for (var i = 0; i < hs.length; i++)
			hs[i].call(this, param);
};

function override(obj, orig, upgrade) {
	var origFunc = obj[orig];
	obj[orig] = function() {
		var args = [].slice.apply(arguments);
		args.unshift(origFunc);
		return upgrade.apply(this, args);
	};
}

// Language mappings and settings
OS.lang = function(phrase) {
	return lang[this.language][phrase];
};
// Overriden by cookie or client-side setting
OS.language = config.DEFAULT_LANG;

OS.red_string = function(ref) {
	var dest, linkClass;
	if (/^>\/watch/.test(ref)) {
		dest = 'https://www.youtube.com/' + ref.slice(2);
		linkClass = 'embed watch';
	}
	else if (/^>\/soundcloud/.test(ref)) {
		dest = 'https://soundcloud.com/' + ref.slice(13);
		linkClass = 'embed soundcloud';
	}
	else if (/^>\/pastebin/.test(ref)) {
		dest = dest = 'https://pastebin.com/' + ref.slice(11);
		linkClass = 'embed pastebin';
	}

	// Linkify >>>/board/ URLs
	var board;
	for (var i = 0; i < config.BOARDS.length; i++) {
		board = config.BOARDS[i];
		if (!new RegExp('^>\\/' + board + '\\/').test(ref))
			continue;
		dest = '../' + board;
		linkClass = '';
		break;
	}

	if (!dest) {
		this.tamashii(parseInt(ref, 10));
		return;
	}
	this.callback(new_tab_link(encodeURI(dest), '>>' + ref, linkClass));
};

OS.break_heart = function(frag) {
	if (frag.safe)
		return this.callback(frag);
	var bits = frag.split(break_re);
	for (var i = 0; i < bits.length; i++) {
		/* anchor refs */
		var morsels = bits[i].split(ref_re);
		for (var j = 0; j < morsels.length; j++) {
			var m = morsels[j];
			if (j % 2)
				this.red_string(m);
			else if (i % 2) {
				this.geimu(m);
				this.callback(safe('<wbr>'));
			}
			else
				this.geimu(m);
		}
	}
};

OS.iku = function(token, to) {
	var state = this.state;
	if (state[0] == DEF.S_QUOTE && to != DEF.S_QUOTE)
		this.callback(safe('</em>'));
	switch(to) {
		case DEF.S_QUOTE:
			if (state[0] != DEF.S_QUOTE) {
				this.callback(safe('<em>'));
				state[0] = DEF.S_QUOTE;
			}
			this.break_heart(token);
			break;
		case DEF.S_SPOIL:
			if (token[1] == '/') {
				state[1]--;
				this.callback(safe('</del>'));
			}
			else {
				var del = {html: '<del>'};
				this.trigger('spoilerTag', del);
				this.callback(safe(del.html));
				state[1]++;
			}
			break;
		default:
			this.break_heart(token);
			break;
	}
	state[0] = to;
};

OS.fragment = function(frag) {
	var chunks = frag.split(/(\[\/?spoiler\])/i);
	var state = this.state;
	for (var i = 0; i < chunks.length; i++) {
		var chunk = chunks[i], q = (state[0] === DEF.S_QUOTE);
		if (i % 2) {
			var to = DEF.S_SPOIL;
			if (chunk[1] == '/' && state[1] < 1)
				to = q ? DEF.S_QUOTE : DEF.S_NORMAL;
			this.iku(chunk, to);
			continue;
		}
		lines = chunk.split(/(\n)/);
		for (var l = 0; l < lines.length; l++) {
			var line = lines[l];
			if (l % 2)
				this.iku(safe('<br>'), DEF.S_BOL);
			else if (state[0] === DEF.S_BOL && line[0] == '>')
				this.iku(line, DEF.S_QUOTE);
			else if (line)
				this.iku(line, q ? DEF.S_QUOTE
					: DEF.S_NORMAL);
		}
	}
};

OS.karada = function(body) {
	var output = [];
	this.state = [DEF.S_BOL, 0];
	this.callback = function(frag) {
		output.push(frag);
	};
	this.fragment(body);
	this.callback = null;
	if (this.state[0] == DEF.S_QUOTE)
		output.push(safe('</em>'));
	for (var i = 0; i < this.state[1]; i++)
		output.push(safe('</del>'));
	return output;
};

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

function serverTime() {
	var d = new Date().getTime();
	// On the server or time difference not compared yet
	if (isNode || !serverTimeOffset)
		return d;
	return d + serverTimeOffset;
}

function readable_dice(bit, d) {
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
}

OS.geimu = function(text) {
	if (!this.dice) {
		this.eLinkify ? this.linkify(text) : this.callback(text);
		return;
	}

	var bits = text.split(dice_re);
	for (var i = 0; i < bits.length; i++) {
		var bit = bits[i];
		if (!(i % 2) || !parse_dice(bit)) {
			this.eLinkify ? this.linkify(bit) : this.callback(bit);
		}
		else if (this.queueRoll) {
			this.queueRoll(bit);
		}
		else if (!this.dice[0]) {
			this.eLinkify ? this.linkify(bit) : this.callback(bit);
		}
		else {
			var d = this.dice.shift();
			this.callback(safe('<strong>'));
			this.strong = true; // for client DOM insertion
			this.callback(readable_dice(bit, d));
			this.strong = false;
			this.callback(safe('</strong>'));
		}
	}
};

OS.linkify = function(text) {

	var bits = text.split(/(https?:\/\/[^\s"<>]*[^\s"<>'.,!?:;])/);
	for (var i = 0; i < bits.length; i++) {
		if (i % 2) {
			var e = escape_html(bits[i]);
			// open in new tab, and disavow target
			this.callback(safe('<a href="' + e +
				'" rel="nofollow" target="_blank">' +
				e + '</a>'));
		}
		else
			this.callback(bits[i]);
	}
};

function chibi(imgnm, src) {
	var name = '', ext = '';
	var m = imgnm.match(/^(.*)(\.\w{3,4})$/);
	if (m) {
		name = m[1];
		ext = m[2];
	}
	var bits = [safe('<a href="'), src, safe('" download="'), imgnm];
	if (name.length >= 38) {
		bits.push(safe('" title="'), imgnm);
		imgnm = [name.slice(0, 30), safe('(&hellip;)'), ext];
	}
	bits.push(safe('" rel="nofollow">'), imgnm, safe('</a>'));
	return bits;
}

OS.spoiler_info = function(index, toppu) {
	var large = toppu;
	var hd = toppu || this.thumbStyle != 'small';
	return {
		thumb: encodeURI(mediaURL + 'kana/spoiler' + (hd ? '' : 's')
			+ index + '.png'),
		dims: large ? imagerConfig.THUMB_DIMENSIONS
			: imagerConfig.PINKY_DIMENSIONS,
	};
};

var spoilerImages = imagerConfig.SPOILER_IMAGES;

function pick_spoiler(metaIndex) {
	var imgs = spoilerImages;
	var n = imgs.length;
	var i;
	if (metaIndex < 0)
		i = Math.floor(Math.random() * n);
	else
		i = metaIndex % n;
	return {index: imgs[i], next: (i + 1) % n};
}
exports.pick_spoiler = pick_spoiler;

function new_tab_link(srcEncoded, inside, cls, brackets) {
	if (brackets)
		inside = '[' + inside + '] ';
	return [safe('<a href="' + srcEncoded + '" target="_blank"' +
			(cls ? ' class="' + cls + '"' : '') +
			' rel="nofollow">'), inside, safe('</a>')];
}


OS.image_paths = function() {
	if (!this._imgPaths) {
		this._imgPaths = {
			src: mediaURL + 'src/',
			thumb: mediaURL + 'thumb/',
			mid: mediaURL + 'mid/',
			vint: mediaURL + 'vint/',
		};
		this.trigger('mediaPaths', this._imgPaths);
	}
	return this._imgPaths;
};

var audioIndicator = "\u266B"; // musical note

OS.gazou = function(info, toppu) {
	var src, caption;
	// TODO: Unify archive and normal thread caption logic
	if (info.vint) {
		src = encodeURI('../outbound/hash/' + info.MD5);
		var google = encodeURI('../outbound/g/' + info.vint);
		var iqdb = encodeURI('../outbound/iqdb/' + info.vint);
		caption = [this.lang('search') + ' ', new_tab_link(google, '[Google]'),
			' ',
			new_tab_link(iqdb, '[iqdb]'), ' ',
			new_tab_link(src, '[foolz]')];
	}
	else {
		var google = encodeURI('../outbound/g/' + info.thumb);
		var iqdb = encodeURI('../outbound/iqdb/' + info.thumb);
		var saucenao = encodeURI('../outbound/sn/' + info.thumb);
		var foolz = encodeURI('../outbound/hash/' + info.MD5);
		var exhentai = encodeURI('../outbound/exh/' + info.SHA1);
		src = encodeURI(this.image_paths().src + info.src);
		caption = [
			new_tab_link(src, (this.thumbStyle == 'hide') ? '[Show]'
				: info.src, 'imageSrc'), ' ',
			new_tab_link(google, 'G', 'imageSearch google', true),
			new_tab_link(iqdb, 'Iq', 'imageSearch iqdb', true),
			new_tab_link(saucenao, 'Sn', 'imageSearch saucenao', true),
			new_tab_link(foolz, 'Fz', 'imageSearch foolz', true),
			new_tab_link(exhentai, 'Ex', 'imageSearch exhentai', true),
		];
	}

	var img = this.gazou_img(info, toppu);
	var size = info.size ? readable_filesize(info.size) + ', ' : '';
	var dims = info.dims[0] + 'x' + info.dims[1];

	// We need da data for da client to walk da podium
	return [safe('<figure data-img="'), (isNode ? escapeJSON(info) : ''),
		safe('"><figcaption>'),
		caption, safe(' <i>('),
		info.audio ? (audioIndicator + ', ') : '',
		info.length ? (info.length + ', ') : '',
		readable_filesize(info.size), ', ',
		dims, (info.apng ? ', APNG' : ''),
		this.full ? [', ', chibi(info.imgnm, img.src)] : '',
		safe(')</i></figcaption>'),
		this.thumbStyle == 'hide' ? '' : img.html,
		safe('</figure>\n\t')];
};

exports.thumbStyles = ['small', 'sharp', 'hide'];

OS.gazou_img = function(info, toppu, href) {
	var src, thumb;
	var imgPaths = this.image_paths();
	var m = info.src ? /.gif$/.test(info.src) : false;
	if (!info.vint)
		src = thumb = encodeURI(imgPaths.src + info.src);

	var d = info.dims;
	var w = d[0], h = d[1], tw = d[2], th = d[3];
	if (info.spoiler && !this.spoilToggle) {
		var sp = this.spoiler_info(info.spoiler, toppu);
		thumb = sp.thumb;
		tw = sp.dims[0];
		th = sp.dims[1];
	}
	else if (info.vint) {
		tw = tw || w;
		th = th || h;
		src = encodeURI('../outbound/hash/' + info.MD5);
		thumb = imgPaths.vint + info.vint;
	}
	else if (m && this.autoGif)
		thumb = src;
	else if (this.thumbStyle == 'sharp' && info.mid)
		thumb = encodeURI(imgPaths.mid + info.mid);
	else if (info.thumb)
		thumb = encodeURI(imgPaths.thumb + info.thumb);
	else {
		tw = w;
		th = h;
	}

	var img = '<img src="' + thumb + '"';
	if (tw && th)
		img += ' width="' + tw + '" height="' + th + '">';
	else
		img += '>';
	if (imagerConfig.IMAGE_HATS)
		img = '<span class="hat"></span>' + img;
	// Override src with href, if specified
	img = new_tab_link(href || src, safe(img));
	return {html: img, src: src};
};

function escapeJSON(obj) {
	return encodeURIComponent(JSON.stringify(obj));
}

function catchJSON(string) {
	return JSON.parse(decodeURIComponent(string));
}

function readable_filesize(size) {
	/* Dealt with it. */
	if (size < 1024)
		return size + ' B';
	if (size < 1048576)
		return Math.round(size / 1024) + ' KB';
	size = Math.round(size / 104857.6).toString();
	return size.slice(0, -1) + '.' + size.slice(-1) + ' MB';
}
exports.readable_filesize = readable_filesize;

function pad(n) {
	return (n < 10 ? '0' : '') + n;
}

OS.readable_time = function(time) {
	var h = this.tz_offset;
	var offset;
	if (h || h == 0)
		offset = h * 60 * 60 * 1000;
	else
		// XXX: would be nice not to construct new Dates all the time
		offset = new Date().getTimezoneOffset() * -60 * 1000;

	return this.readableDate(new Date(time + offset));
};

OS.readableDate = function(d) {
	return pad(d.getUTCDate()) + ' ' + this.lang('year')[d.getUTCMonth()] + ' '
		+ d.getUTCFullYear() + '(' + this.lang('week')[d.getUTCDay()] + ')'
		+ pad(d.getUTCHours()) + ':' + pad(d.getUTCMinutes());
};

// Readable elapsed time since post
OS.relative_time = function(then, now) {
	var min = Math.floor((now - then) / (60 * 1000)),
		ago = this.lang('ago');
	if (min < 1)
		return this.lang('just_now');
	if (min < 60)
		return ago(min, this.lang('unit_minute'));
	var hours = Math.floor(min / 60);
	if (hours < 24)
		return ago(hours, this.lang('unit_hour'));
	var days = Math.floor(hours / 24);
	if (days < 30)
		return ago(days, this.lang('unit_day'));
	var months = Math.floor(days / 30);
	if (months < 12)
		return ago(months, this.lang('unit_month'));
	return ago(Math.floor(months / 12), this.lang('unit_year'));
};

function datetime(time) {
	var d = new Date(time);
	return (d.getUTCFullYear() + '-' + pad(d.getUTCMonth() + 1) + '-' +
		pad(d.getUTCDate()) + 'T' + pad(d.getUTCHours()) + ':' +
		pad(d.getUTCMinutes()) + ':' + pad(d.getUTCSeconds()) + 'Z');
}

OS.post_url = function(num, op, quote) {
	op = op || num;
	return (this.op == op ? '' : op) + (quote ? '#q' : '#') + num;
};

OS.post_ref = function(num, op, desc_html) {
	var ref = '&gt;&gt;' + num;
	if (desc_html)
		ref += ' ' + desc_html;
	else if (this.op && this.op != op)
		ref += ' \u2192';
	else if (num == op && this.op == op)
		ref += ' (OP)';
	return safe('<a href="' + this.post_url(num, op, false) + '">' + ref
		+ '</a>');
};

OS.post_nav = function(post) {
	var n = post.num, o = post.op;
	return safe('<nav><a href="' + this.post_url(n, o, false) +
		'">No.</a><a href="' + this.post_url(n, o, true) +
		'">' + n + '</a></nav>');
};

function action_link_html(href, name, id) {
	return '<span class="act"><a href="' + href + '"'
		+ (id ? ' id="' + id + '"' : '')
		+ '>' + name + '</a></span>';
}
exports.action_link_html = action_link_html;

reasonable_last_n = function(n) {
	return Number.isInteger(n) && n >= 5 && n <= 500;
};
exports.reasonable_last_n = reasonable_last_n;

OS.expansion_links_html = function(num) {
	return ' &nbsp; ' + action_link_html(num, this.lang('expand')) + ' '
		+ action_link_html(num + '?last=' + this.lastN, this.lang('last')
			+ '&nbsp;' + this.lastN);
};

OS.atama = function(data) {
	var auth = data.auth;
	var header = auth ? [safe('<b class="'), auth.toLowerCase(), safe('">')]
		: [safe('<b>')];
	if (data.subject)
		header.unshift(safe('<h3>「'), data.subject, safe('」</h3> '));
	if (data.name || !data.trip) {
		header.push(data.name || this.lang('anon'));
		if (data.trip)
			header.push(' ');
	}
	if (data.trip)
		header.push(safe('<code>' + data.trip + '</code>'));
	if (auth)
		header.push(' ## ' + (auth == 'Admin' ? hotConfig.ADMIN_ALIAS
			: hotConfig.MOD_ALIAS));
	this.trigger('headerName', {header: header, data: data});
	header.push(safe('</b>'));
	if (data.email) {
		header.unshift(safe('<a class="email" href="mailto:'
			+ encodeURI(data.email) + '" target="_blank">'));
		header.push(safe('</a>'));
	}
	// Format according to client's relative post timestamp setting
	var title = this.rTime ? this.readable_time(data.time) : '';
	var text = this.rTime ? this.relative_time(data.time, new Date().getTime())
		: this.readable_time(data.time);
	header.push(safe(' <time datetime="' + datetime(data.time) + '"' +
		'title="' + title + '"' +
		'>' + text + '</time> '),
		this.post_nav(data));
	if (!this.full && !data.op) {
		var ex = this.expansion_links_html(data.num);
		header.push(safe(ex));
	}
	this.trigger('headerFinish', {header: header, data: data});
	header.unshift(safe('<header>'));
	header.push(safe('</header>\n\t'));
	return header;
};

OS.monogatari = function(data, toppu) {
	var tale = {header: this.atama(data)};
	this.dice = data.dice;
	var body = this.karada(data.body);
	tale.body = [safe(
			'<blockquote' +
			(isNode ? ' data-body="' + escapeJSON(data.body) + '"' : '') + '>'),
		body, safe('</blockquote>'
			)];
	if (data.image && !data.hideimg)
		tale.image = this.gazou(data.image, toppu);
	return tale;
};

OS.mono = function(data) {
	var info = {
		data: data,
		classes: data.editing ? ['editing'] : [],
		style: ''
	};
	this.trigger('openArticle', info);
	var cls = info.classes.length && info.classes.join(' '),
		o = safe('\t<article id="' + data.num + '"' +
			(cls ? ' class="' + cls + '"' : '') +
			(info.style ? ' style="' + info.style + '"' : '') +
			'>'),
		c = safe('</article>\n'),
		gen = this.monogatari(data, false);
	return flatten([o, gen.header, gen.image || '', gen.body, c]).join('');
};

OS.monomono = function(data, cls) {
	if (data.locked)
		cls = cls ? cls + ' locked' : 'locked';
	var style;
	var o = safe('<section id="' + data.num +
		(cls ? '" class="' + cls : '') +
		(style ? '" style="' + style : '') +
		'" data-sync="' + (data.hctr || 0) +
		(data.full ? '' : '" data-imgs="' + data.imgctr) + '">'),
		c = safe('</section>\n'),
		gen = this.monogatari(data, true);
	return flatten([o, gen.image || '', gen.header, gen.body, '\n', c]);
};

parse_name = function(name) {
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
	return [name.substr(0, 100), tripcode.substr(0, 128),
		secure.substr(0, 128)];
};
exports.parse_name = parse_name;

function random_id() {
	return Math.floor(Math.random() * 1e16) + 1;
}
