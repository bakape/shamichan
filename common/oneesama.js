/*
 Core rendering object both on the client and server
 */

'use strict';

var imports = require('./imports'),
	index = require('./index'),
	util = require('./util');

const config = imports.config,
	flatten = util.flatten,
	new_tab_link = util.new_tab_link,
	pad = util.pad,
	parseHTML = util.parseHTML,
	safe = util.safe;

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
OS.lang = imports.isNode ? imports.lang[config.DEFAULT_LANG].common
	: imports.lang;

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
	for (let i = 0, len = config.BOARDS.length; i < len; i++) {
		let board = config.BOARDS[i];
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
	let bits = frag.split(break_re);
	for (let i = 0, len = bits.length; i < len; i++) {
		/* anchor refs */
		const morsels = bits[i].split(ref_re);
		for (let j = 0, len = morsels.length; j < len; j++) {
			const m = morsels[j];
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
	let state = this.state;
	if (state[0] == index.S_QUOTE && to != index.S_QUOTE)
		this.callback(safe('</em>'));
	switch(to) {
		case index.S_QUOTE:
			if (state[0] != index.S_QUOTE) {
				this.callback(safe('<em>'));
				state[0] = index.S_QUOTE;
			}
			this.break_heart(token);
			break;
		case index.S_SPOIL:
			if (token[1] == '/') {
				state[1]--;
				this.callback(safe('</del>'));
			}
			else {
				const del = {html: '<del>'};
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
	const chunks = frag.split(/(\[\/?spoiler])/i);
	let state = this.state;
	const	q = state[0] === index.S_QUOTE;
	for (let i = 0, len = chunks.length; i < len; i++) {
		let chunk = chunks[i];
		if (i % 2) {
			let to = index.S_SPOIL;
			if (chunk[1] == '/' && state[1] < 1)
				to = q ? index.S_QUOTE : index.S_NORMAL;
			this.iku(chunk, to);
			continue;
		}
		const lines = chunk.split(/(\n)/);
		for (let l = 0, len = lines.length; l < len; l++) {
			const line = lines[l];
			if (l % 2)
				this.iku(safe('<br>'), index.S_BOL);
			else if (state[0] === index.S_BOL && line[0] == '>')
				this.iku(line, index.S_QUOTE);
			else if (line)
				this.iku(line, q ? index.S_QUOTE : index.S_NORMAL);
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
		output.push(safe('</em>'));
	for (let i = 0; i < this.state[1]; i++)
		output.push(safe('</del>'));
	return output;
};

OS.geimu = function(text) {
	if (!this.dice) {
		this.eLinkify ? this.linkify(text) : this.callback(text);
		return;
	}

	const bits = text.split(util.dice_re);
	for (let i = 0, len = bits.length; i < len; i++) {
		const bit = bits[i];
		if (!(i % 2) || !util.parse_dice(bit))
			this.eLinkify ? this.linkify(bit) : this.callback(bit);
		else if (this.queueRoll)
			this.queueRoll(bit);
		else if (!this.dice[0])
			this.eLinkify ? this.linkify(bit) : this.callback(bit);
		else {
			let d = this.dice.shift();
			this.callback(safe('<strong>'));
			this.strong = true; // for client DOM insertion
			this.callback(util.readable_dice(bit, d));
			this.strong = false;
			this.callback(safe('</strong>'));
		}
	}
};

OS.linkify = function(text) {

	let bits = text.split(/(https?:\/\/[^\s"<>]*[^\s"<>'.,!?:;])/);
	for (let i = 0, len = bits.length; i < len; i++) {
		if (i % 2) {
			const e = util.escape_html(bits[i]);
			// open in new tab, and disavow target
			this.callback(safe(
				`<a href="${e}" rel="nofollow" target="_blank">${e}</a>`
			));
		}
		else
			this.callback(bits[i]);
	}
};

OS.spoiler_info = function(ind, toppu) {
	var large = toppu;
	var hd = toppu || this.thumbStyle != 'small';
	return {
		thumb: encodeURI(config.MEDIA_URL + 'spoil/spoiler' + (hd
				? ''
				: 's')
			+ ind + '.png'),
		dims: large ? config.THUMB_DIMENSIONS
			: config.PINKY_DIMENSIONS
	};
};

OS.image_paths = function() {
	if (!this._imgPaths) {
		var mediaURL = config.MEDIA_URL;
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
			this.lang.search + ' ', new_tab_link(google, '[Google]'), ' ',
			new_tab_link(iqdb, '[iqdb]'), ' ',
			new_tab_link(src, '[foolz]')
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
			new_tab_link(src, (this.thumbStyle == 'hide') ? '[Show]'
				: info.src, 'imageSrc'), ' ',
			new_tab_link(google, 'G', 'imageSearch google', true),
			new_tab_link(iqdb, 'Iq', 'imageSearch iqdb', true),
			new_tab_link(saucenao, 'Sn', 'imageSearch saucenao', true),
			new_tab_link(foolz, 'Fz', 'imageSearch foolz', true),
			new_tab_link(exhentai, 'Ex', 'imageSearch exhentai', true)
		];
	}

	var img = this.gazou_img(info, toppu);
	var dims = info.dims[0] + 'x' + info.dims[1];

	// We need da data for da client to walk da podium
	return [
		safe('<figure><figcaption>'),
		caption, safe(' <i>('),
		info.audio ? ("\u266B" + ', ') : '', // musical note
		info.length ? (info.length + ', ') : '',
		util.readable_filesize(info.size), ', ',
		dims, (info.apng ? ', APNG' : ''),
		this.full ? [', ', chibi(info.imgnm, img.src)] : '',
		safe(')</i></figcaption>'),
		this.thumbStyle == 'hide' ? '' : img.html,
		safe('</figure>\n\t')
	];
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
	if (config.IMAGE_HATS)
		img = '<span class="hat"></span>' + img;
	// Override src with href, if specified
	img = new_tab_link(href || src, safe(img));
	return {html: img, src: src};
};

OS.post_url = function(num, op) {
	op = op || num;
	return `${this.op == op ? '' : op}#${num}`;
};

OS.post_ref = function(num, op, desc_html) {
	var ref = '&gt;&gt;' + num;
	if (desc_html)
		ref += ' ' + desc_html;
	else if (this.op && this.op != op)
		ref += ' \u2192';
	else if (num == op && this.op == op)
		ref += ' (OP)';
	return safe(
		`<a href="${this.post_url(num, op)}" class="history">${ref}</a>`
	);
};

OS.post_nav = function(post) {
	const n = post.num;
	var o = post.op;
	return safe(parseHTML
		`<nav>
			<a href="${this.post_url(n, o)}" class="history">No.</a>
			<a href="${this.post_url(n, o)}" class="quote">${n}</a>
		</nav>`
	);
};

OS.expansion_links_html = function(num) {
	return ' &nbsp; '
		+ util.action_link_html(num, this.lang.expand, null, 'history')
		+ ' '
		+ util.action_link_html(`${num}?last=${this.lastN}`,
			`${this.lang.last}&nbsp;${this.lastN}`,
			null,
			'history'
		);
};

OS.atama = function(data) {
	var auth = data.auth;
	var header = auth ? [
		safe('<b class="'),
		auth.toLowerCase(),
		safe('">')
	]
		: [safe('<b>')];
	if (data.subject)
		header.unshift(safe('<h3>「'), data.subject, safe('」</h3> '));
	if (data.name || !data.trip) {
		header.push(data.name || this.lang.anon);
		if (data.trip)
			header.push(' ');
	}
	if (data.trip)
		header.push(safe(`<code>'${data.trip}</code>`));
	if (auth)
		header.push(' ## ' + (auth == 'Admin' ? imports.hotConfig.ADMIN_ALIAS
				: imports.hotConfig.MOD_ALIAS));
	this.trigger('headerName', {header: header, data: data});
	header.push(safe('</b>'));
	if (data.email) {
		header.unshift(safe(parseHTML
			`<a class="email"
				href="mailto:${encodeURI(data.email)}"
				target="_blank">
			</a>`
		));
	}
	header.push(' ', safe(this.time(data.time)), ' ', this.post_nav(data));
	if (!this.full && !data.op) {
		var ex = this.expansion_links_html(data.num);
		header.push(safe(ex));
	}
	this.trigger('headerFinish', {header: header, data: data});
	header.unshift(safe('<header>'));
	header.push(safe('</header>\n\t'));
	return header;
};

OS.time = function(time) {
	// Format according to client's relative post timestamp setting
	var title, text;
	if (this.rTime) {
		title = this.readable_time(time);
		text = this.relative_time(time, Date.now());
	}
	else {
		title = '';
		text = this.readable_time(time);
	}
	return parseHTML
		`<time datetime="${datetime(time)}" title="${title}">
			${text}
		</time>`;
};

function datetime(time) {
	var d = new Date(time);
	return (d.getUTCFullYear() + '-' + pad(d.getUTCMonth() + 1) + '-'
	+ pad(d.getUTCDate()) + 'T' + pad(d.getUTCHours()) + ':'
	+ pad(d.getUTCMinutes()) + ':' + pad(d.getUTCSeconds()) + 'Z');
}

OS.readable_time = function(time) {
	var h = this.tz_offset;
	var offset;
	if (h || h == 0)
		offset = h * 60 * 60 * 1000;
	else
	// XXX: would be nice not to construct new Dates all the time
		offset = new Date().getTimezoneOffset() * -60 * 1000;
	var d = new Date(time + offset);

	return parseHTML
		`${pad(d.getUTCDate())} ${this.lang.year[d.getUTCMonth()]}
		 ${d.getUTCFullYear()}(${this.lang.week[d.getUTCDay()]})
		${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
};

// Readable elapsed time since post
OS.relative_time = function(then, now) {
	let time = Math.floor((now - then) / 60000);
	if (time < 1)
		return this.lang.just_now;

	const divide = [60, 24, 30, 12],
		unit = ['minute', 'hour', 'day', 'month'];
	for (let i = 0, len = divide.length; i < len; i++) {
		if (time < divide[i])
			return this.lang.ago(time, this.lang['unit_' + unit[i]]);
		time = Math.floor(time / divide[i]);
	}

	return this.lang.ago(time, this.lang.unit_year);
};

OS.monogatari = function(data, toppu) {
	var tale = {header: this.atama(data)};
	this.dice = data.dice;
	var body = this.karada(data.body);
	tale.body = [
		safe('<blockquote>'),
		body,
		safe('</blockquote>')
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
	var o = safe(`<section id="${data.num}"`
			+ (cls ? ` class="${cls}"` : '') + '>'),
		c = safe('</section>\n'),
		gen = this.monogatari(data, true);
	return flatten([o, gen.image || '', gen.header, gen.body, '\n', c]);
};

OS.replyBox = function() {
	return `<aside class="act"><a>${this.lang.reply}</a></aside>`;
};

OS.newThreadBox = function() {
	return `<aside class="act"><a>${this.lang.newThread}</a></aside>`;
};
