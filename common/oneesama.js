/*
 Core rendering object both on the client and server
 */

var index = require('./index');

var state, config, hotConfig, imagerConfig, lang, main;
if (index.isNode) {
	state = require('../server/state');
	config = require('../config');
	hotConfig = state.hot;
	imagerConfig = require('../imager/config');
	lang = require('../lang/');
}
else {
	main = require('../alpha/main');
	state = require('../alpha/state');
	config = state.config.attributes;
	hotConfig = state.hotConfig.attributes;
	imagerConfig = state.imagerConfig.attributes;
	lang = main.lang;
}

var OneeSama = function(t) {
	this.tamashii = t;
	this.hooks = {};
};
module.exports = OneeSama;

var OS = OneeSama.prototype;

var break_re = new RegExp("(\\S{" + index.WORD_LENGTH_LIMIT + "})");

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
		this.hooks[name] = [func];
	else if (hs.indexOf(func) < 0)
		hs.push(func);
};

OS.trigger = function(name, param) {
	var hs = this.hooks[name];
	if (hs)
		for (var i = 0; i < hs.length; i++)
			hs[i].call(this, param);
};

/*
 * Language mappings and settings. Overriden by cookie server-side and
 * bootstraped into the template client-side
 */
OS.lang = index.isNode ? lang[config.DEFAULT_LANG].common : lang;

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
		dest = 'https://pastebin.com/' + ref.slice(11);
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
	this.callback(index.new_tab_link(encodeURI(dest), '>>' + ref, linkClass));
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
				this.callback(index.safe('<wbr>'));
			}
			else
				this.geimu(m);
		}
	}
};

OS.iku = function(token, to) {
	var state = this.state;
	if (state[0] == index.S_QUOTE && to != index.S_QUOTE)
		this.callback(index.safe('</em>'));
	switch(to) {
		case index.S_QUOTE:
			if (state[0] != index.S_QUOTE) {
				this.callback(index.safe('<em>'));
				state[0] = index.S_QUOTE;
			}
			this.break_heart(token);
			break;
		case index.S_SPOIL:
			if (token[1] == '/') {
				state[1]--;
				this.callback(index.safe('</del>'));
			}
			else {
				var del = {html: '<del>'};
				this.trigger('spoilerTag', del);
				this.callback(index.safe(del.html));
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
	var chunks = frag.split(/(\[\/?spoiler])/i);
	var state = this.state;
	for (var i = 0; i < chunks.length; i++) {
		var chunk = chunks[i], q = (state[0] === index.S_QUOTE);
		if (i % 2) {
			var to = index.S_SPOIL;
			if (chunk[1] == '/' && state[1] < 1)
				to = q ? index.S_QUOTE : index.S_NORMAL;
			this.iku(chunk, to);
			continue;
		}
		var lines = chunk.split(/(\n)/);
		for (var l = 0; l < lines.length; l++) {
			var line = lines[l];
			if (l % 2)
				this.iku(index.safe('<br>'), index.S_BOL);
			else if (state[0] === index.S_BOL && line[0] == '>')
				this.iku(line, index.S_QUOTE);
			else if (line)
				this.iku(line, q ? index.S_QUOTE
					: index.S_NORMAL);
		}
	}
};

OS.karada = function(body) {
	var output = [];
	this.state = [index.S_BOL, 0];
	this.callback = function(frag) {
		output.push(frag);
	};
	this.fragment(body);
	this.callback = null;
	if (this.state[0] == index.S_QUOTE)
		output.push(index.safe('</em>'));
	for (var i = 0; i < this.state[1]; i++)
		output.push(index.safe('</del>'));
	return output;
};

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
			this.callback(index.safe('<strong>'));
			this.strong = true; // for client DOM insertion
			this.callback(index.readable_dice(bit, d));
			this.strong = false;
			this.callback(index.safe('</strong>'));
		}
	}
};

OS.linkify = function(text) {

	var bits = text.split(/(https?:\/\/[^\s"<>]*[^\s"<>'.,!?:;])/);
	for (var i = 0; i < bits.length; i++) {
		if (i % 2) {
			var e = escape_html(bits[i]);
			// open in new tab, and disavow target
			this.callback(index.safe('<a href="' + e +
				'" rel="nofollow" target="_blank">' +
				e + '</a>'));
		}
		else
			this.callback(bits[i]);
	}
};

OS.spoiler_info = function(ind, toppu) {
	var large = toppu;
	var hd = toppu || this.thumbStyle != 'small';
	return {
		thumb: encodeURI(imagerConfig.MEDIA_URL + 'spoil/spoiler' + (hd ? '' : 's')
			+ ind + '.png'),
		dims: large ? imagerConfig.THUMB_DIMENSIONS
			: imagerConfig.PINKY_DIMENSIONS
	};
};

OS.image_paths = function() {
	if (!this._imgPaths) {
		var mediaURL = imagerConfig.MEDIA_URL;
		this._imgPaths = {
			src: mediaURL + 'src/',
			thumb: mediaURL + 'thumb/',
			mid: mediaURL + 'mid/',
			vint: mediaURL + 'vint/'
		};
		this.trigger('mediaPaths', this._imgPaths);
	}
	return this._imgPaths;
};

OS.gazou = function(info, toppu) {
	var src, caption;
	// TODO: Unify archive and normal thread caption logic
	var google, iqdb;
	if (info.vint) {
		src = encodeURI('../outbound/hash/' + info.MD5);
		google = encodeURI('../outbound/g/' + info.vint);
		iqdb = encodeURI('../outbound/iqdb/' + info.vint);
		caption = [
			this.lang.search + ' ', index.new_tab_link(google, '[Google]'), ' ',
			index.new_tab_link(iqdb, '[iqdb]'), ' ',
			index.new_tab_link(src, '[foolz]')
		];
	}
	else {
		google = encodeURI('../outbound/g/' + info.thumb);
		iqdb = encodeURI('../outbound/iqdb/' + info.thumb);
		var saucenao = encodeURI('../outbound/sn/' + info.thumb);
		var foolz = encodeURI('../outbound/hash/' + info.MD5);
		var exhentai = encodeURI('../outbound/exh/' + info.SHA1);
		src = encodeURI(this.image_paths().src + info.src);
		caption = [
			index.new_tab_link(src, (this.thumbStyle == 'hide') ? '[Show]'
				: info.src, 'imageSrc'), ' ',
			index.new_tab_link(google, 'G', 'imageSearch google', true),
			index.new_tab_link(iqdb, 'Iq', 'imageSearch iqdb', true),
			index.new_tab_link(saucenao, 'Sn', 'imageSearch saucenao', true),
			index.new_tab_link(foolz, 'Fz', 'imageSearch foolz', true),
			index.new_tab_link(exhentai, 'Ex', 'imageSearch exhentai', true)
		];
	}

	var img = this.gazou_img(info, toppu);
	var dims = info.dims[0] + 'x' + info.dims[1];

	// We need da data for da client to walk da podium
	return [
		index.safe('<figure data-img="'), (index.isNode ? escapeJSON(info) : ''),
		index.safe('"><figcaption>'),
		caption, index.safe(' <i>('),
		info.audio ? ("\u266B" + ', ') : '', // musical note
		info.length ? (info.length + ', ') : '',
		index.readable_filesize(info.size), ', ',
		dims, (info.apng ? ', APNG' : ''),
		this.full ? [', ', chibi(info.imgnm, img.src)] : '',
		index.safe(')</i></figcaption>'),
		this.thumbStyle == 'hide' ? '' : img.html,
		index.safe('</figure>\n\t')
	];
};

function escapeJSON(obj) {
	return encodeURIComponent(JSON.stringify(obj));
}

function chibi(imgnm, src) {
	var name = '', ext = '';
	var m = imgnm.match(/^(.*)(\.\w{3,4})$/);
	if (m) {
		name = m[1];
		ext = m[2];
	}
	var bits = [index.safe('<a href="'), src, index.safe('" download="'), imgnm];
	if (name.length >= 38) {
		bits.push(index.safe('" title="'), imgnm);
		imgnm = [name.slice(0, 30), index.safe('(&hellip;)'), ext];
	}
	bits.push(index.safe('" rel="nofollow">'), imgnm, index.safe('</a>'));
	return bits;
}

OS.gazou_img = function(info, toppu, href) {
	var src, thumb;
	var imgPaths = this.image_paths();
	var m = info.src ? /.gif$/.test(info.src) : false;
	if (!info.vint)
		src = thumb = encodeURI(imgPaths.src + info.src);

	var d = info.dims;
	var w = d[0], h = d[1], tw = d[2], th = d[3];
	if (info.spoiler && this.spoilToggle) {
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
	img = index.new_tab_link(href || src, index.safe(img));
	return {html: img, src: src};
};

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
	return index.pad(d.getUTCDate()) + ' ' + this.lang.year[d.getUTCMonth()] + ' '
		+ d.getUTCFullYear() + '(' + this.lang.week[d.getUTCDay()] + ')'
		+ index.pad(d.getUTCHours()) + ':' + index.pad(d.getUTCMinutes());
};

// Readable elapsed time since post
OS.relative_time = function(then, now) {
	var min = Math.floor((now - then) / (60 * 1000)),
		ago = this.lang.ago;
	if (min < 1)
		return this.lang.just_now;
	if (min < 60)
		return ago(min, this.lang.unit_minute);
	var hours = Math.floor(min / 60);
	if (hours < 24)
		return ago(hours, this.lang.unit_hour);
	var days = Math.floor(hours / 24);
	if (days < 30)
		return ago(days, this.lang.unit_day);
	var months = Math.floor(days / 30);
	if (months < 12)
		return ago(months, this.lang.unit_month);
	return ago(Math.floor(months / 12), this.lang.unit_year);
};

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
	return index.safe('<a href="' + this.post_url(num, op, false) + '"'
		+ ' class="history">' + ref + '</a>');
};

OS.post_nav = function(post) {
	var n = post.num, o = post.op;
	return index.safe('<nav><a href="' + this.post_url(n, o, false) +
		'">No.</a><a href="' + this.post_url(n, o, true) +
		'">' + n + '</a></nav>');
};

OS.expansion_links_html = function(num) {
	return ' &nbsp; ' + index.action_link_html(num, this.lang.expand, null,
			'history')
		+ ' '
		+ index.action_link_html(num + '?last=' + this.lastN,
			this.lang.last + '&nbsp;' + this.lastN, null, 'history');
};

OS.atama = function(data) {
	var auth = data.auth;
	var header = auth ? [
		index.safe('<b class="'),
		auth.toLowerCase(),
		index.safe('">')
	]
		: [index.safe('<b>')];
	if (data.subject)
		header.unshift(index.safe('<h3>「'), data.subject, index.safe('」</h3> '));
	if (data.name || !data.trip) {
		header.push(data.name || this.lang.anon);
		if (data.trip)
			header.push(' ');
	}
	if (data.trip)
		header.push(index.safe('<code>' + data.trip + '</code>'));
	if (auth)
		header.push(' ## ' + (auth == 'Admin' ? hotConfig.ADMIN_ALIAS
				: hotConfig.MOD_ALIAS));
	this.trigger('headerName', {header: header, data: data});
	header.push(index.safe('</b>'));
	if (data.email) {
		header.unshift(index.safe('<a class="email" href="mailto:'
			+ encodeURI(data.email) + '" target="_blank">'));
		header.push(index.safe('</a>'));
	}
	// Format according to client's relative post timestamp setting
	var title = this.rTime ? this.readable_time(data.time) : '';
	var text = this.rTime ? this.relative_time(data.time, new Date().getTime())
		: this.readable_time(data.time);
	header.push(index.safe(' <time datetime="' + datetime(data.time) + '"' +
			'title="' + title + '"' +
			'>' + text + '</time> '),
		this.post_nav(data));
	if (!this.full && !data.op) {
		var ex = this.expansion_links_html(data.num);
		header.push(index.safe(ex));
	}
	this.trigger('headerFinish', {header: header, data: data});
	header.unshift(index.safe('<header>'));
	header.push(index.safe('</header>\n\t'));
	return header;
};

function datetime(time) {
	var d = new Date(time);
	return (d.getUTCFullYear() + '-' + index.pad(d.getUTCMonth() + 1) + '-'
	+ index.pad(d.getUTCDate()) + 'T' + index.pad(d.getUTCHours()) + ':'
	+ index.pad(d.getUTCMinutes()) + ':' + index.pad(d.getUTCSeconds()) + 'Z');
}

OS.monogatari = function(data, toppu) {
	var tale = {header: this.atama(data)};
	this.dice = data.dice;
	var body = this.karada(data.body);
	tale.body = [
		index.safe(
			'<blockquote' +
			(index.isNode ? ' data-body="' + escapeJSON(data.body) + '"'
				: '') + '>'),
		body, index.safe('</blockquote>'
		)
	];
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
		o = index.safe('\t<article id="' + data.num + '"' +
			(cls ? ' class="' + cls + '"' : '') +
			(info.style ? ' style="' + info.style + '"' : '') +
			'>'),
		c = index.safe('</article>\n'),
		gen = this.monogatari(data, false);
	return index.flatten([o, gen.header, gen.image || '', gen.body, c]).join('');
};

OS.monomono = function(data, cls) {
	if (data.locked)
		cls = cls ? cls + ' locked' : 'locked';
	var o = index.safe('<section id="' + data.num +
			(cls ? '" class="' + cls : '') +
			'" data-sync="' + (data.hctr || 0) +
			(data.full ? '' : '" data-imgs="' + data.imgctr) + '">'),
		c = index.safe('</section>\n'),
		gen = this.monogatari(data, true);
	return index.flatten([o, gen.image || '', gen.header, gen.body, '\n', c]);
};

OS.replyBox = function() {
	return '<aside class="act"><a>' + this.lang.reply + '</a></aside>';
};

OS.newThreadBox = function() {
	return '<aside class="act"><a>' + this.lang.newThread + '</a></aside>';
};
