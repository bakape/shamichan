var _ = require('./lib/underscore');
var config = require('./config');
var imagerConfig = require('./imager/config');
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
DEFINES.LOCK_THREAD = 11;
DEFINES.UNLOCK_THREAD = 12;

DEFINES.IMAGE_STATUS = 31;
DEFINES.SYNCHRONIZE = 32;
DEFINES.EXECUTE_JS = 33;
DEFINES.MOVE_THREAD = 34;
DEFINES.UPDATE_BANNER = 35;
DEFINES.TEARDOWN = 36;

DEFINES.MODEL_SET = 50;
DEFINES.COLLECTION_RESET = 55;
DEFINES.COLLECTION_ADD = 56;
DEFINES.SUBSCRIBE = 60;
DEFINES.UNSUBSCRIBE = 61;

DEFINES.ANON = 'Anonymous';
DEFINES.INPUT_ROOM = 20;
DEFINES.MAX_POST_LINES = 30;
DEFINES.MAX_POST_CHARS = 2000;
DEFINES.WORD_LENGTH_LIMIT = 120;

DEFINES.S_NORMAL = 0;
DEFINES.S_BOL = 1;
DEFINES.S_QUOTE = 2;
DEFINES.S_SPOIL = 3;

var mediaURL = imagerConfig.MEDIA_URL;

function is_pubsub(t) {
	return t > 0 && t < 30;
}
exports.is_pubsub = is_pubsub;

function FSM(start) {
	this.state = start;
	this.spec = {acts: {}, ons: {}, wilds: {}, preflights: {}};
}
exports.FSM = FSM;

FSM.prototype.clone = function () {
	var second = new FSM(this.state);
	second.spec = this.spec;
	return second;
};

// Handlers on arriving to a new state
FSM.prototype.on = function (key, f) {
	var ons = this.spec.ons[key];
	if (ons)
		ons.push(f);
	else
		this.spec.ons[key] = [f];
	return this;
};

// Sanity checks before attempting a transition
FSM.prototype.preflight = function (key, f) {
	var pres = this.spec.preflights[key];
	if (pres)
		pres.push(f);
	else
		this.spec.preflights[key] = [f];
};

// Specify transitions and an optional handler function
FSM.prototype.act = function (trans_spec, on_func) {
	var halves = trans_spec.split('->');
	if (halves.length != 2)
		throw new Error("Bad FSM spec: " + trans_spec);
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

FSM.prototype.feed = function (ev, param) {
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
		if (_.isArray(frag))
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

function is_noko(email) {
	return email && email.indexOf('@') == -1 && /noko/i.test(email);
}
exports.is_noko = is_noko;
function is_sage(email) {
	return config.SAGE_ENABLED && email &&
			email.indexOf('@') == -1 && /sage/i.test(email);
}
exports.is_sage = is_sage;

var OneeSama = function (t) {
	this.tamashii = t;
	this.hooks = {};
};
exports.OneeSama = OneeSama;
var OS = OneeSama.prototype;

var break_re = new RegExp("(\\S{" + DEFINES.WORD_LENGTH_LIMIT + "})");
/* internal refs, embeds */
var ref_re = />>(\d+|>\/watch\?v=[\w-]{11}(?:#t=[\dhms]{1,9})?|>\/soundcloud\/[\w-]{1,40}\/[\w-]{1,80}|>\/(?:a|foolz)\/\d{0,10})/;

OS.hook = function (name, func) {
	var hs = this.hooks[name];
	if (!hs)
		this.hooks[name] = hs = [func];
	else if (hs.indexOf(func) < 0)
		hs.push(func);
};

OS.trigger = function (name, param) {
	var hs = this.hooks[name];
	if (hs)
		for (var i = 0; i < hs.length; i++)
			hs[i].call(this, param);
};

function override(obj, orig, upgrade) {
	var origFunc = obj[orig];
	obj[orig] = function () {
		var args = [].slice.apply(arguments);
		args.unshift(origFunc);
		return upgrade.apply(this, args);
	};
}

OS.red_string = function (ref) {
	var prefix = ref.slice(0, 3);
	var dest, linkClass;
	if (prefix == '>/w') {
		dest = 'http://www.youtube.com/' + ref.slice(2);
		linkClass = 'embed watch';
	}
	else if (prefix == '>/s') {
		dest = 'http://soundcloud.com/' + ref.slice(13);
		linkClass = 'embed soundcloud';
	}
	else if (prefix == '>/a') {
		var num = parseInt(ref.slice(4), 10);
		dest = '../outbound/a/' + (num ? ''+num : '');
	}
	else if (prefix == '>/f') {
		var num = parseInt(ref.slice(8), 10);
		dest = '../outbound/foolz/' + (num ? ''+num : '');
	}
	else {
		this.tamashii(parseInt(ref, 10));
		return;
	}
	this.callback(new_tab_link(encodeURI(dest), '>>' + ref, linkClass));
};

OS.break_heart = function (frag) {
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

OS.spoiler_info = function (index, toppu) {
	var large = toppu || this.thumbStyle == 'large';
	var hd = toppu || this.thumbStyle != 'small';
	return {
		thumb: encodeURI(mediaURL + 'kana/spoiler' + (hd ? '' : 's')
				+ index + '.png'),
		dims: large ? imagerConfig.THUMB_DIMENSIONS
				: imagerConfig.PINKY_DIMENSIONS,
	};
};

function pick_spoiler(metaIndex) {
	var imgs = imagerConfig.SPOILER_IMAGES;
	var n = imgs.normal.length;
	var count = n + imgs.trans.length;
	var i;
	if (metaIndex < 0)
		i = Math.floor(Math.random() * count);
	else
		i = metaIndex % count;
	var spoiler = i < n ? imgs.normal[i] : imgs.trans[i - n];
	return {index: spoiler, next: (i+1) % count};
}
exports.pick_spoiler = pick_spoiler;

function new_tab_link(srcEncoded, inside, cls) {
	return [safe('<a href="' + srcEncoded + '" target="_blank"' +
		(cls ? ' class="'+cls+'"' : '') +
		' rel="nofollow">'), inside, safe('</a>')];
}

var imgPaths = {
	src: mediaURL + 'src/',
	thumb: mediaURL + 'thumb/',
	mid: mediaURL + 'mid/',
	vint: mediaURL + 'vint/',
};

OS.gazou = function (info, toppu) {
	var src, name, caption;
	this.trigger('mediaPaths', imgPaths);
	if (info.vint) {
		src = encodeURI('../outbound/hash/' + info.MD5);
		var google = encodeURI('../outbound/g/' + info.vint);
		var iqdb = encodeURI('../outbound/iqdb/' + info.vint);
		caption = ['Search ', new_tab_link(google, '[Google]'), ' ',
			new_tab_link(iqdb, '[iqdb]'), ' ',
			new_tab_link(src, '[foolz]')];
	}
	else {
		src = encodeURI(imgPaths.src + info.src);
		caption = ['Image ', new_tab_link(src, info.src)];
	}

	var img = this.thumbStyle=='hide' ? '' : this.gazou_img(info, toppu);
	var dims = info.dims[0] + 'x' + info.dims[1];

	return [safe('<figure data-MD5="'), info.MD5, safe('"><figcaption>'),
		caption, safe(' <i>('), readable_filesize(info.size) + ', ',
		dims, (info.apng ? ', APNG' : ''),
		this.full ? chibi(info.imgnm) : '', safe(')</i></figcaption>'),
		img, safe('</figure>\n\t')];
};

exports.thumbStyles = ['small', 'sharp', 'large', 'hide'];

OS.gazou_img = function (info, toppu) {
	var src, thumb;
	if (!info.vint)
		src = thumb = encodeURI(imgPaths.src + info.src);

	var d = info.dims;
	var w = d[0], h = d[1], tw = d[2], th = d[3];
	if (info.spoiler) {
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
	else if (this.thumbStyle != 'small' && info.mid) {
		thumb = encodeURI(imgPaths.mid + info.mid);
		if (!toppu && this.thumbStyle == 'large') {
			tw *= 2;
			th *= 2;
		}
	}
	else if (info.thumb)
		thumb = encodeURI(imgPaths.thumb + info.thumb);
	else {
		tw = w;
		th = h;
	}

	var img = '<img src="'+thumb+'"';
	if (tw && th)
		img += ' width="' +tw+'" height="'+th+'">';
	else
		img += '>';
	if (imagerConfig.IMAGE_HATS)
		img = '<span class="hat"></span>' + img;
	img = new_tab_link(src, safe(img));
	return img;
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
	var k = "日月火水木金土"[d.getUTCDay()];
	return (d.getUTCFullYear() + '/' + pad(d.getUTCMonth()+1) + '/' +
		pad(d.getUTCDate()) + '(' + k + ') ' +
		pad(d.getUTCHours()) + ':' +
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
	var ref = '&gt;&gt;' + num;
	if (this.op && this.op != op)
		ref += ' \u2192';
	else if (num == op && this.op == op)
		ref += ' (OP)';
	return safe('<a href="'+this.post_url(num, op, false)+'">'+ref+'</a>');
};

OS.post_nav = function (post) {
	var n = post.num, o = post.op;
	return safe('<nav><a href="' + this.post_url(n, o, false) +
			'">No.</a><a href="' + this.post_url(n, o, true) +
			'">' + n + '</a></nav>');
};

function action_link_html(href, name) {
	return '<span class="act"><a href="'+href+'">'+name+'</a></span>';
}
exports.action_link_html = action_link_html;

function last_n_html(num) {
	return action_link_html('THREAD?lastN', 'Last&nbsp;N').replace(
		/N/g, config.THREAD_LAST_N).replace('THREAD', num);
}

function expansion_links_html(num, omit) {
	var html = ' &nbsp; ' + action_link_html(num, 'Expand');
	if (omit > config.THREAD_LAST_N)
		html += ' ' + last_n_html(num);
	return html;
}

OS.atama = function (data) {
	var auth = data.auth;
	var header = auth ? [safe('<b class="'),auth.toLowerCase(),safe('">')]
			: [safe('<b>')];
	if (data.subject)
		header.push(safe('<h3>「'), data.subject, safe('」</h3> '));
	header.push(data.name || DEFINES.ANON);
	if (data.trip)
		header.push(safe(' <code>' + data.trip + '</code>'));
	if (auth)
		header.push(' ## ' + auth);
	this.trigger('headerName', {header: header, data: data});
	header.push(safe('</b>'));
	if (data.email) {
		header.unshift(safe('<a class="email" href="mailto:'
				+ encodeURI(data.email) + '">'));
		header.push(safe('</a>'));
	}
	header.push(safe(' <time datetime="' + datetime(data.time) +
			'">' + readable_time(data.time) + '</time> '),
			this.post_nav(data));
	if (!this.full && !data.op)
		header.push(safe(expansion_links_html(data.num, data.omit)));
	this.trigger('headerFinish', {header: header, data: data});
	header.unshift(safe('<header>'));
	header.push(safe('</header>\n\t'));
	return header;
};

OS.monogatari = function (data, toppu) {
	var tale = {header: this.atama(data)};
	this.dice = data.dice;
	var body = this.karada(data.body);
	tale.body = [safe('<blockquote>'), body, safe('</blockquote>')];
	if (data.image && !data.hideimg)
		tale.image = this.gazou(data.image, toppu);
	return tale;
};

OS.mono = function (data) {
	var info = {
		data: data,
		classes: data.editing ? ['editing'] : [],
		style: ''
	};
	this.trigger('openArticle', info);
	var cls = info.classes.length && info.classes.join(' '),
	    o = safe('\t<article id="'+data.num+'"' +
			(cls ? ' class="'+cls+'"' : '') +
			(info.style ? ' style="'+info.style+'"' : '') +
			'>'),
	    c = safe('</article>\n'),
	    gen = this.monogatari(data, false);
	return flatten([o, gen.header, gen.image || '', gen.body, c]).join('');
};

OS.monomono = function (data, cls) {
	if (data.locked)
		cls = cls ? cls+' locked' : 'locked';
	var o = safe('<section id="' + data.num +
		(cls ? '" class="' + cls : '') +
		'" data-sync="' + (data.hctr || 0) +
		(data.full ? '' : '" data-imgs="'+data.imgctr) + '">'),
	    c = safe('</section>\n'),
	    gen = this.monogatari(data, true);
	return flatten([o, gen.image || '', gen.header, gen.body, '\n', c]);
};

function pluralize(n, noun) {
	return n + ' ' + noun + (n == 1 ? '' : 's');
}
exports.pluralize = pluralize;

exports.abbrev_msg = function (omit, img_omit) {
	return omit + (omit==1 ? ' reply' : ' replies') + (img_omit
		? ' and ' + pluralize(img_omit, 'image')
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
