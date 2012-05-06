var config = require('./config');
var DEFINES = exports;
DEFINES.INVALID = 0;

DEFINES.INSERT_POST = 2;
DEFINES.UPDATE_POST = 3;
DEFINES.FINISH_POST = 4;
DEFINES.CATCH_UP = 5;
DEFINES.INSERT_IMAGE = 6;
DEFINES.SPOILER_IMAGES = 7;
DEFINES.DELETE_IMAGES = 8;
DEFINES.DELETE_POSTS = 9;
DEFINES.DELETE_THREAD = 10;

DEFINES.ALLOCATE_POST = 30;
DEFINES.IMAGE_STATUS = 31;
DEFINES.SYNCHRONIZE = 32;
DEFINES.EXECUTE_JS = 33;
DEFINES.MOVE_THREAD = 34;
DEFINES.UPDATE_BANNER = 35;

DEFINES.ANON = 'Anonymous';
DEFINES.INPUT_ROOM = 20;
DEFINES.MAX_POST_LINES = 30;
DEFINES.MAX_POST_CHARS = 2000;
DEFINES.WORD_LENGTH_LIMIT = 120;

DEFINES.S_NORMAL = 0;
DEFINES.S_BOL = 1;
DEFINES.S_QUOTE = 2;
DEFINES.S_SPOIL = 3;

var mediaURL = config.MEDIA_URL;

function is_pubsub(t) {
	return t >= DEFINES.INSERT_POST && t <= DEFINES.DELETE_THREAD;
}
exports.is_pubsub = is_pubsub;

function FSM(start) {
	this.state = start;
	this.acts = {};
	this.ons = {};
	this.wilds = {};
	this.preflights = {};
}
exports.FSM = FSM;

// Handlers on arriving to a new state
FSM.prototype.on = function (key, f) {
	var ons = this.ons[key];
	if (ons)
		ons.push(f);
	else
		this.ons[key] = [f];
	return this;
};

// Sanity checks before attempting a transition
FSM.prototype.preflight = function (key, f) {
	var pres = this.preflights[key];
	if (pres)
		pres.push(f);
	else
		this.preflights[key] = [f];
};

// Specify transitions and an optional handler function
FSM.prototype.act = function (spec, on_func) {
	var halves = spec.split('->');
	if (halves.length != 2)
		throw new Error("Bad FSM spec: " + spec);
	var parts = halves[0].split(',');
	var dest = halves[1].match(/^\s*(\w+)\s*$/)[1];
	var tok;
	for (var i = parts.length-1; i >= 0; i--) {
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
			this.wilds[tok] = dest;
		else {
			var acts = this.acts[src];
			if (!acts)
				this.acts[src] = acts = {};
			acts[tok] = dest;
		}
	}
	if (on_func)
		this.on(dest, on_func);
	return this;
};

FSM.prototype.feed = function (ev, param) {
	var from = this.state, acts = this.acts[from];
	var to = (acts && acts[ev]) || this.wilds[ev];
	if (to && from != to) {
		var ps = this.preflights[to];
		for (var i = 0; ps && i < ps.length; i++)
			if (!ps[i].call(this, param))
				return false;
		this.state = to;
		var fs = this.ons[to];
		for (var i = 0; fs && i < fs.length; i++)
			fs[i].call(this, param);
	}
	return true;
};

FSM.prototype.feeder = function (ev) {
	var self = this;
	return function (param) {
		self.feed(ev, param);
	};
};

var entities = {'&' : '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;'};
function escape_html(html) {
	return html.replace(/[&<>"]/g, function (c) {
		return entities[c];
	});
}
exports.escape_html = escape_html;

function escape_fragment(frag) {
	var t = typeof(frag);
	if (t == 'object' && frag && typeof(frag.safe) == 'string')
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
		if (frag.constructor == Array)
			out = out.concat(flatten(frag));
		else
			out.push(escape_fragment(frag));
	}
	return out;
}

function safe(frag) {
	return {safe: frag};
}
exports.safe = safe;

function map_unsafe(frags, func) {
	for (var i = 0; i < frags.length; i++) {
		if (typeof(frags[i]) == 'string')
			frags[i] = func(frags[i]);
		else if (frags[i].constructor == Array)
			frags[i] = map_unsafe(frags[i], func);
	}
	return frags;
}

function is_noko(email) {
	return email && email.indexOf('@') == -1 && email.match(/noko/i);
}
exports.is_noko = is_noko;
function is_sage(email) {
	return config.SAGE_ENABLED && email &&
			email.indexOf('@') == -1 && email.match(/sage/i);
}
exports.is_sage = is_sage;

var OneeSama = function (t) {
	this.tamashii = t;
	this.hooks = {};
};
exports.OneeSama = OneeSama;
var OS = OneeSama.prototype;

var break_re = new RegExp("(\\S{" + DEFINES.WORD_LENGTH_LIMIT + "})");
/* internal refs and youtube videos */
var ref_re = />>(\d+|>\/?(?:watch\?)?v[=\/][\w-]{11}(?:#t=[\dhms]{1,9})?)/;
var youtube_re = /^>>>\/?(?:watch\?)?v[=\/]([\w-]{11})(#t=[\dhms]{1,9})?$/;
var youtube_time_re = /^#t=(?:(\d\d?)h)?(?:(\d\d?)m)?(?:(\d\d?)s)?$/;
var youtube_url_re = /(?:>>>*?)?(?:http:\/\/)?(?:www\.)?youtube\.com\/watch\?((?:[^\s#&=]+=[^\s#&]*&)*)?v=([\w-]{11})((?:&[^\s#&=]+=[^\s#&]*)*)&?(#t=[\dhms]{1,9})?/;

OS.hook = function (name, func) {
	var hs = this.hooks[name];
	if (!hs)
		this.hooks[name] = hs = [func];
	else if (hs.indexOf(func) < 0)
		hs.push(func);
};

OS.trigger = function (name, param, context) {
	var hs = this.hooks[name];
	if (hs)
		for (var i = 0; i < hs.length; i++)
			param = hs[i].call(this, param, context);
	return param;
};

function override(obj, orig, upgrade) {
	var origFunc = obj[orig];
	obj[orig] = function () {
		var args = [].slice.apply(arguments);
		args.unshift(origFunc);
		return upgrade.apply(this, args);
	};
}

OS.break_heart = function (frag) {
	if (frag.safe)
		return this.callback(frag);
	var bits = frag.split(break_re);
	for (var i = 0; i < bits.length; i++) {
		/* anchor refs */
		var morsels = bits[i].split(ref_re);
		for (var j = 0; j < morsels.length; j++) {
			var m = morsels[j];
			if (j % 2) {
				if (m[0] == '>') {
					/* This is alright since it's always
					   a single word */
					this.callback(safe('<cite>' +
							escape_html('>>' + m) +
							'</cite>'));
				}
				else
					this.tamashii(parseInt(m, 10));
			}
			else if (i % 2) {
				this.geimu(m);
				this.callback(safe('<wbr>'));
			}
			else
				this.geimu(m);
		}
	}
};

OS.iku = function (token, to) {
	var state = this.state;
	if (state[0] == DEFINES.S_QUOTE && to != DEFINES.S_QUOTE)
		this.callback(safe('</em>'));
	switch (to) {
	case DEFINES.S_QUOTE:
		if (state[0] != DEFINES.S_QUOTE) {
			this.callback(safe('<em>'));
			state[0] = DEFINES.S_QUOTE;
		}
		this.break_heart(token);
		break;
	case DEFINES.S_SPOIL:
		if (token[1] == '/') {
			state[1]--;
			this.callback(safe('</del>'));
		}
		else {
			this.callback(safe('<del>'));
			state[1]++;
		}
		break;
	default:
		this.break_heart(token);
		break;
	}
	state[0] = to;
}

OS.fragment = function (frag) {
	var chunks = frag.split(/(\[\/?spoiler\])/i);
	var state = this.state;
	for (var i = 0; i < chunks.length; i++) {
		var chunk = chunks[i], q = (state[0] === DEFINES.S_QUOTE);
		if (i % 2) {
			var to = DEFINES.S_SPOIL;
			if (chunk[1] == '/' && state[1] < 1)
				to = q ? DEFINES.S_QUOTE : DEFINES.S_NORMAL;
			this.iku(chunk, to);
			continue;
		}
		lines = chunk.split(/(\n)/);
		for (var l = 0; l < lines.length; l++) {
			var line = lines[l];
			if (l % 2)
				this.iku(safe('<br>'), DEFINES.S_BOL);
			else if (state[0] === DEFINES.S_BOL && line[0] == '>')
				this.iku(line, DEFINES.S_QUOTE);
			else if (line)
				this.iku(line, q ? DEFINES.S_QUOTE
						: DEFINES.S_NORMAL);
		}
	}
};

OS.karada = function (body) {
	var output = [];
	this.state = [DEFINES.S_BOL, 0];
	this.callback = function (frag) { output.push(frag); }
	this.fragment(body);
	this.callback = null;
	if (this.state[0] == DEFINES.S_QUOTE)
		output.push(safe('</em>'));
	for (var i = 0; i < this.state[1]; i++)
		output.push(safe('</del>'));
	return output;
}

var dice_re = /(#flip|#\d{0,2}d\d{1,4}(?:[+-]\d{1,4})?)/i;
exports.dice_re = dice_re;

function parse_dice(frag) {
	if (frag == '#flip')
		return {n: 1, faces: 2};
	var m = frag.match(/^#(\d*)d(\d+)([+-]\d+)?$/i);
	if (!m)
		return false;
	var n = parseInt(m[1], 10) || 1, faces = parseInt(m[2], 10);
	if (n < 1 || n > 10 || faces < 2 || faces > 100)
		return false;
	var info = {n: n, faces: faces};
	if (m[3])
		info.bias = parseInt(m[3], 10);
	return info;
}
exports.parse_dice = parse_dice;

function readable_dice(bit, d) {
	if (bit == '#flip')
		return '#flip (' + (d[1] == 2) + ')';
	var f = d[0], n = d.length, b = 0;
	if (d[n-1] && typeof d[n-1] == 'object') {
		b = d[n-1].bias;
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

OS.geimu = function (text) {
	if (!this.dice)
		return this.callback(text);
	var bits = text.split(dice_re);
	for (var i = 0, x = 0; i < bits.length; i++) {
		var bit = bits[i];
		if (!(i % 2) || !parse_dice(bit)) {
			this.callback(bit);
		}
		else if (this.queueRoll) {
			this.queueRoll(bit);
		}
		else if (!this.dice[0]) {
			this.callback(bit);
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

function chibi(text) {
	var m = text.match(/^(.{30}).{8,}(\.\w{3,4})$/);
	/* Comma inlined for convience in OS.gazou (beware of concatenating
	 * lists with strings) */
	if (!m)
		return ', ' + text;
	return [safe(', <abbr title="'), text, safe('">'), m[1],
		safe('(&hellip;)'), m[2], safe('</abbr>')];
}

function spoiler_info(index, toppu) {
	return {
		thumb: encodeURI(mediaURL + 'kana/spoiler' + (toppu ? '' : 's')
				+ index + '.png'),
		dims: toppu ? config.THUMB_DIMENSIONS
				: config.PINKY_DIMENSIONS,
	};
}

OS.gazou = function (info, toppu) {
	var src, thumb, name;
	if (info.vint) {
		src = encodeURI('../outbound/' + info.MD5);
		thumb = mediaURL + 'vint/' + info.vint;
		srcname = info.MD5;
	}
	else {
		src = thumb = encodeURI(mediaURL + 'src/' + info.src);
		srcname = info.src;
	}
	var d = info.dims;
	var w = d[0], h = d[1], tw = d[2], th = d[3];
	if (info.spoiler) {
		var sp = spoiler_info(info.spoiler, toppu);
		thumb = sp.thumb;
		tw = sp.dims[0];
		th = sp.dims[1];
	}
	else if (info.vint) {
		tw = tw || w;
		th = th || h;
	}
	else if (info.thumb)
		thumb = encodeURI(mediaURL + 'thumb/' + info.thumb);
	else {
		tw = w;
		th = h;
	}
	return [safe('<figure data-MD5="'), info.MD5, safe('"><figcaption>' +
		'Image <a href="' + src + '" target="_blank">'), srcname,
		safe('</a> <i>(' + readable_filesize(info.size) + ', ' +
		w + 'x' + h), info.apng ? ', APNG' : '',
		this.full ? chibi(info.imgnm) : '',
		safe(')</i></figcaption><a href="'+src+'" target="_blank">' +
		'<img src="' + thumb + '" width="' +
		tw + '" height="' + th + '"></a>' + '</figure>\n\t')];
};

function readable_filesize(size) {
       /* Metric. Deal with it. */
       if (size < 1000)
               return size + ' B';
       if (size < 1000000)
               return Math.round(size / 1000) + ' KB';
       size = Math.round(size / 100000).toString();
       return size.slice(0, -1) + '.' + size.slice(-1) + ' MB';
}

function pad(n) {
	return (n < 10 ? '0' : '') + n;
}

function readable_time(time) {
	var d = new Date(time - new Date().getTimezoneOffset() * 60000);
	return (d.getUTCFullYear() + '/' + pad(d.getUTCMonth()+1) + '/' +
		pad(d.getUTCDate()) + ' ' + pad(d.getUTCHours()) + ':' +
		pad(d.getUTCMinutes()));
}
exports.readable_time = readable_time;

function datetime(time) {
	var d = new Date(time);
	return (d.getUTCFullYear() + '-' + pad(d.getUTCMonth()+1) + '-' +
		pad(d.getUTCDate()) + 'T' + pad(d.getUTCHours()) + ':' +
		pad(d.getUTCMinutes()) + ':' + pad(d.getUTCSeconds()) + 'Z');
}

OS.post_url = function (num, op, quote) {
	op = op || num;
	return (this.op == op ? '' : op) + (quote ? '#q' : '#') + num;
};

OS.post_ref = function (num, op) {
	return safe('<a href="' + this.post_url(num, op, false)
			+ '">&gt;&gt;' + num + '</a>');
};

OS.post_nav = function (post) {
	var n = post.num, o = post.op;
	return safe('<nav><a href="' + this.post_url(n, o, false) +
			'">No.</a><a href="' + this.post_url(n, o, true) +
			'">' + n + '</a></nav>');
};

var lastNfrag = '?lastN">Last&nbsp;N</a></span>'.replace(/N/g,
		config.THREAD_LAST_N);
function expand_html(num, omit) {
	var html = ' &nbsp; <span class="act"><a href="' + num +
			'">Expand</a></span>';
	if (omit > config.THREAD_LAST_N)
		html += ' <span class="act"><a href="' + num + lastNfrag;
	return html;
}

OS.atama = function (data) {
	var auth = data.auth;
	var header = auth ? [safe('<b class="'),auth.toLowerCase(),safe('">')]
			: [safe('<b>')];
	header.push(data.name || DEFINES.ANON);
	if (data.trip)
		header.push(safe(' <code>' + data.trip + '</code>'));
	if (auth)
		header.push(' ## ' + auth);
	header = this.trigger('header', header, data);
	header.push(safe('</b>'));
	if (data.email) {
		header.unshift(safe('<a class="email" href="mailto:'
				+ encodeURI(data.email) + '">'));
		header.push(safe('</a>'));
	}
	header.unshift(safe('<header>'));
	header.push(safe(' <time pubdate datetime="' + datetime(data.time) +
			'">' + readable_time(data.time) + '</time> '),
			this.post_nav(data));
	if (!this.full && !data.op)
		header.push(safe(expand_html(data.num, data.omit)));
	header.push(safe('</header>\n\t'));
	return header;
};

OS.monogatari = function (data, t) {
	var header = this.atama(data);
	this.dice = data.dice;
	var body = this.karada(data.body);
	body = [safe('<blockquote>'), body, safe('</blockquote>')];
	if (!data.image || data.hideimg)
		return {header: header, body: body};
	return {header: header, image: this.gazou(data.image, t), body: body};
};

OS.mono = function (data) {
	var o = safe(data.editing
			? '\t<article id="' + data.num + '" class="editing">'
			: '\t<article id="' + data.num + '">'),
	    c = safe('</article>\n'),
	    gen = this.monogatari(data, false);
	return flatten([o, gen.header, gen.image || '', gen.body, c]).join('');
};

OS.monomono = function (data, cls) {
	var o = safe('<section id="' + data.num +
		(cls ? '" class="' + cls : '') +
		'" data-sync="' + (data.hctr || 0) +
		(data.full ? '' : '" data-imgs="'+data.imgctr) + '">'),
	    c = safe('</section>\n'),
	    gen = this.monogatari(data, true);
	return flatten([o, gen.image || '', gen.header, gen.body, '\n', c]);
};

exports.abbrev_msg = function (omit, img_omit) {
	return omit + (omit==1 ? ' reply' : ' replies') + (img_omit
		? ' and ' + img_omit + ' image' + (img_omit==1 ? '' : 's')
		: '') + ' omitted.';
};

exports.parse_name = function (name) {
	var tripcode = '', secure = '';
	var hash = name.indexOf('#');
	if (hash >= 0) {
		tripcode = name.substr(hash+1);
		name = name.substr(0, hash);
		hash = tripcode.indexOf('#');
		if (hash >= 0) {
			secure = escape_html(tripcode.substr(hash+1));
			tripcode = tripcode.substr(0, hash);
		}
		tripcode = escape_html(tripcode);
	}
	name = name.trim().replace(config.EXCLUDE_REGEXP, '');
	return [name.substr(0, 100), tripcode.substr(0, 128),
			secure.substr(0, 128)];
};

exports.random_id = function () {
	return Math.floor(Math.random() * 1e16) + 1;
};
