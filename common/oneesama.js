/*
 Rendering singleton both server and client-side
 */

const _ = require('underscore'),
	imports = require('./imports'),
	index = require('./index'),
	util = require('./util'),
	{config} = imports,
	{flatten, join, pad, parseHTML, safe} = util,
	escape = util.escape_html;

const break_re = new RegExp("(\\S{" + index.WORD_LENGTH_LIMIT + "})");

// `>>>/${link}/` referal links and embeds
const ref_re = (function () {
	let ref_re = String.raw`>>(\d+|>\/watch\?v=[\w-]{11}(?:#t=[\dhms]{1,9})?
		|>\/soundcloud\/[\w-]{1,40}\/[\w-]{1,80}|>\/pastebin\/\w+`
			.replace(/[\n\t]+/gm, '');

	for (let board in config.link_targets) {
		ref_re += String.raw`|>\/${board}\/(?:\w+\/?)?`;
	}

	ref_re += ')';
	return new RegExp(ref_re);
})();

// Generate the static part of image search links
const searchBase = (function() {
	const models = [
		{
			class: 'google',
			url: 'https://www.google.com/searchbyimage?image_url=',
			type: 'thumb',
			symbol: 'G'
		},
		{
			class: 'iqdb',
			url: 'http://iqdb.org/?url=',
			type: 'thumb',
			symbol: 'Iq'
		},
		{
			class: 'saucenao',
			url: 'http://saucenao.com/search.php?db=999&url=',
			type: 'thumb',
			symbol: 'Sn'
		},
		{
			class: 'desustorage',
			type: 'MD5',
			url: 'https://desuarchive.org/_/search/image/',
			symbol: 'Ds'
		},
		{
			class: 'exhentai',
			type: 'SHA1',
			url: 'http://exhentai.org/?fs_similar=1&fs_exp=1&f_shash=',
			symbol: 'Ex'
		}
	];

	let base = [];
	for (let i = 0, l = models.length; i < l; i++) {
		let model = models[i];
		base[i] = [
			parseHTML
			`<a target="_blank"
					rel="nofollow"
					class="imageSearch ${model.class}"
					href="${model.url}`,
			model.type,
			`">${model.symbol}</a>`
		];
	}
	return base;
})();

class OneeSama {
	constructor(args) {
		_.extend(this, args);
		this.hooks = {};
	}
	hook(name, func) {
		let hs = this.hooks[name];
		if (!hs)
			this.hooks[name] = [func];
		else if (hs.indexOf(func) < 0)
			hs.push(func);
	}
	trigger(name, param) {
		let hs = this.hooks[name];
		if (!hs)
			return;
		for (var i = 0; i < hs.length; i++)
			hs[i].call(this, param);
	}
	// Render OP
	section(data, cls) {
		cls = cls || '';
		if (data.locked)
			cls = cls ? cls + ' locked' : 'locked';
		const gen = this.monogatari(data);
		return flatten([
			safe(`\n<section id="${data.num}" class="${cls}">`),
			safe(gen.image),
			safe(gen.header),
			gen.body,
			safe('</section>\n')
		]);
	}
	// Render reply
	article(data) {
		const gen = this.monogatari(data),
			cls = data.editing ? 'editing' : '';
		return join([
			safe(`\n\t<article id="${data.num}" class="${cls}">`),
			safe(gen.header),
			safe(gen.image),
			gen.body,
			safe('</article>')
		]);
	}
	// Render common post components
	monogatari(data) {
		const tale = {header: this.header(data)};

		// Shallow copy, as to not modify Backbone model values
		this.dice = data.dice && data.dice.slice();
		tale.body = [
			safe('<blockquote>'),
			this.body(data.body),
			safe(`</blockquote><small>${this.backlinks(data.backlinks)}</small>`)
		];
		if (data.mod)
			tale.body.unshift(safe(this.modInfo(data.mod)));
		if (data.banned)
			tale.body.push(safe(this.banned()));
		const {image} = data;
		if (image) {
			// Larger thumbnails for thread images
			image.large = !data.op;
			tale.image = this.image(image);
		}
		else
			tale.image = '';

		return tale;
	}
	header(data) {
		return parseHTML
			`<header>
				<input type="checkbox" class="postCheckbox">
				<span class=control></span>
				${data.subject && `<h3>「${escape(data.subject)}」</h3>`}
				${this.name(data)}
				${this.time(data.time)}
				${this.postNavigation(data)}
				${!this.full && !data.op && this.expansionLinks(data.num)}
			</header>`;
	}
	name(data) {
		let html = '<b class="name';
		const {auth, email} = data;
		if (auth)
			html += ` ${auth === 'admin' ? 'admin' : 'moderator'}`;
		html += '">';
		if (email) {
			html += parseHTML `<a ${{
				class: 'email',
				href: 'mailto:' + encodeURI(email),
				target: 'blank'
			}}>`;
		}
		html += this.resolveName(data);
		if (email)
			html += '</a>';
		html += '</b>';
		if (data.mnemonic)
			html += ' ' + this.mnemonic(data.mnemonic);
		return html;
	}
	resolveName(data) {
		let html = '';
		const {trip, name, auth} = data;
		if (name || !trip) {
			if (name)
				html += escape(name);
			else
				html += this.lang.anon;
			if(trip)
				html += ' ';
		}
		if (trip)
			html += `<code>${escape(trip)}</code>`;
		if (auth)
			html += ` ## ${imports.hotConfig.staff_aliases[auth] || auth}`;
		return html;
	}
	time(time) {
		// Format according to client's relative post timestamp setting
		let title, text;
		const readable = this.readableTime(time);
		if (this.rTime) {
			title = readable;
			text = this.relativeTime(time, Date.now());
		}
		return parseHTML
			`<time title="${title}">
				${text || readable}
			</time>`;
	}
	readableTime(time) {
		let d = new Date(time);
		return pad(d.getDate()) + ' '
			+ this.lang.year[d.getMonth()] + ' '
			+ d.getFullYear()
			+ `(${this.lang.week[d.getDay()]})`
			+`${pad(d.getHours())}:${pad(d.getMinutes())}`;
	}
	readableUTCTime(d, seconds) {
		let html = pad(d.getUTCDate()) + ' '
			+ this.lang.year[d.getUTCMonth()] + ' '
			+ d.getUTCFullYear()
			+ `(${this.lang.week[d.getUTCDay()]})`
			+`${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
		if (seconds)
			html += `:${pad(d.getUTCSeconds())}`;
		html += ' UTC';
		return html;
	}
	// Readable elapsed time since post
	relativeTime(then, now) {
		let time = Math.floor((now - then) / 60000),
			isFuture;
		if (time < 1) {
			// Assume to be client clock imprecission
			if (time > -5)
				return this.lang.just_now;
			else {
				isFuture = true;
				time = -time;
			}
		}

		const divide = [60, 24, 30, 12],
			unit = ['minute', 'hour', 'day', 'month'];
		for (let i = 0; i < divide.length; i++) {
			if (time < divide[i])
				return this.lang.ago(time, this.lang['unit_' + unit[i]],
					isFuture);
			time = Math.floor(time / divide[i]);
		}

		return this.lang.ago(time, this.lang.unit_year, isFuture);
	}
	mnemonic(mnem) {
		return `<b class="mod addr">${mnem}</b>`;
	}
	postNavigation(post) {
		const num = post.num,
			op = post.op;
		return parseHTML
			`<nav>
				<a href="${this.postURL(num, op)}" class="history">
					No.
				</a>
				<a href="${this.postURL(num, op)}" class="quote">
					${num}
				</a>
			</nav>`;
	}
	postURL(num, op) {
		op = op || num;
		return `${this.op == op ? '' : op}#${num}`;
	}
	expansionLinks(num) {
		return parseHTML
			`<span class="act expansionLinks">
				<a href="${num}" class="history">
					${this.lang.expand}
				</a>
				] [
				<a href="${num}?last=${this.lastN}" class="history">
					${this.lang.last} ${this.lastN}
				</a>
			</span>`;
	}
	// Append moderation information. Only exposed to authenticated staff.
	modInfo(info) {
		let html = '<b class="modLog admin">';
		for (let action of info) {
			html += `${this.lang.mod.formatLog(action)}<br>`;
		}
		html += '</b>';
		return html;
	}
	banned() {
		return `<b class="admin banMessage">${this.lang.mod.banMessage}</b>`;
	}
	// Render full blockqoute contents
	body(body) {
		let output = [];
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
	}
	// Parse commited blockqoute fragment
	fragment(frag) {
		const chunks = frag.split(/(\[\/?spoiler])/i);
		let state = this.state;
		const q = state[0] === index.S_QUOTE;
		for (let i = 0, len = chunks.length; i < len; i++) {
			let chunk = chunks[i];
			if (i % 2) {
				let to = index.S_SPOIL;
				if (chunk[1] == '/' && state[1] < 1)
					to = q ? index.S_QUOTE : index.S_NORMAL;
				this.resolveState(chunk, to);
				continue;
			}
			const lines = chunk.split(/(\n)/);
			for (let o = 0, l = lines.length; o < l; o++) {
				const line = lines[o];
				if (o % 2)
					this.resolveState(safe('<br>'), index.S_BOL);
				else if (state[0] === index.S_BOL && line[0] == '>')
					this.resolveState(line, index.S_QUOTE);
				else if (line)
					this.resolveState(line, q ? index.S_QUOTE : index.S_NORMAL);
			}
		}
	}
	// Determine spoiler and quote state
	resolveState(token, to) {
		let state = this.state;
		if (state[0] == index.S_QUOTE && to != index.S_QUOTE)
			this.callback(safe('</em>'));
		switch(to) {
			case index.S_QUOTE:
				if (state[0] != index.S_QUOTE) {
					this.callback(safe('<em>'));
					state[0] = index.S_QUOTE;
				}
				this.breakHeart(token);
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
				this.breakHeart(token);
				break;
		}
		state[0] = to;
	}
	// Split remaning fragment and pass bits to the respective handlers
	breakHeart(frag) {
		if (frag.safe)
			return this.callback(frag);
		let bits = frag.split(break_re);
		for (let i = 0, len = bits.length; i < len; i++) {
			// anchor refs
			const morsels = bits[i].split(ref_re);
			for (let j = 0, l = morsels.length; j < l; j++) {
				const m = morsels[j];
				if (j % 2)
					this.redString(m);
				else if (i % 2) {
					this.parseHashes(m);
					this.callback(safe('<wbr>'));
				}
				else
					this.parseHashes(m);
			}
		}
	}
	// Resolve internal and external URL references
	redString(ref) {
		let dest, linkClass;
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

		// Linkify other `>>>/${link}/` URLs
		for (let link in config.link_targets) {
			const m = ref.match(new RegExp(String.raw`^>\/(${link})\/(\w+\/?)?`));
			if (!m)
				continue;
			dest = config.link_targets[link];
			if (m[2])
				dest += m[2];
			break;
		}

		if (!dest)
			return this.tamashii(parseInt(ref, 10));

		const attrs = {
			href: encodeURI(dest),
			target: '_blank',
			rel: 'nofollow',
			class: linkClass
		};
		this.callback(safe(parseHTML
			`<a ${attrs}>
				>>${escape(ref)}
			</a>`));
	}
	// Render hash commands
	parseHashes(text) {
		if (!this.dice)
			return this.eLinkify ? this.linkify(text) : this.callback(text);

		const bits = text.split(util.dice_re);
		for (let i = 0; i < bits.length; i++) {
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
	}
	// Render external URLs as links
	linkify(text) {
		let bits = text.split(/(https?:\/\/[^\s"<>]*[^\s"<>'.,!?:;])/);
		for (let i = 0, len = bits.length; i < len; i++) {
			if (i % 2) {
				const e = escape(bits[i]);
				// open in new tab, and disalow target
				this.callback(safe(
					`<a href="${e}" rel="nofollow" target="_blank">${e}</a>`
				));
			}
			else
				this.callback(bits[i]);
		}
	}
	backlinks(links) {
		if (!links)
			return '';
		let html = '';
		for (let num in links) {
			if (html)
				html += ' ';
			html += this.postRef(num, links[num]).safe;
		}
		return html;
	}
	// Central image rendering method
	image(data, reveal) {
		const showThumb = this.thumbStyle !== 'hide' || reveal;
		return parseHTML
			`<figure>
				${this.figcaption(data, reveal)}
				${showThumb && config.IMAGE_HATS && '<span class="hat"></span>'}
				${showThumb && this.thumbnail(data)}
			</figure>`;
	}
	// Image header
	figcaption(data, reveal) {
		const list = util.commaList([
			data.audio && '\u266B',
			data.length,
			util.readable_filesize(data.size),
			`${data.dims[0]}x${data.dims[1]}`,
			data.apng && 'APNG'
		]);
		return parseHTML
			`<figcaption>
				${this.thumbStyle === 'hide' && this.hiddenToggle(reveal)}
				${this.imageSearch(data)}
				<i>
					(${list})
					 ${this.imageLink(data)}
				</i>
			</figcaption>`;
	}
	hiddenToggle(reveal) {
		return parseHTML
			`<a class="imageToggle">
				[${this.lang[reveal ? 'hide' : 'show']}]
			</a>`;
	}
	imageSearch(data) {
		let html = '';
		let base = searchBase;
		// Only render google for PDFs and MP3s
		if (['.pdf', '.mp3'].indexOf(data.ext) > -1)
			base = [base[0]];
		// Only use HTTP for thumbnail image search, because IQDB and
		// Saucenao can't into certain SSL cyphers
		const imageURl = this.thumbPath(data)
		for (let i = 0, l = base.length; i < l; i++) {
			let parts = base[i];
			html += parts[0]
				+ encodeURI(parts[1] !== 'thumb' ?  data[parts[1]] : imageURl)
				+ parts[2];
		}

		return html;
	}
	// Get thumbnail path, even if no thumbnail generated
	thumbPath(data, mid) {
		let type = 'thumb';
		if (mid && data.mid)
			type = 'mid';
		else if (!data.thumb)
			type = 'src';

		return this.imagePaths()[type] + data[type];
	}
	imagePaths() {
		if (!this._imgPaths) {
			const mediaURL = config.MEDIA_URL;
			this._imgPaths = {
				src: mediaURL + 'src/',
				thumb: mediaURL + 'thumb/',
				mid: mediaURL + 'mid/',
				spoil: mediaURL + 'spoil/spoiler'
			};
		}
		return this._imgPaths;
	}
	imageLink(data) {
		let name = '',
			imgnm = data.imgnm;
		const m = imgnm.match(/^(.*)\.\w{3,4}$/);
		if (m)
			name = m[1];
		const fullName = escape(imgnm),
			tooLong = name.length >= 38;
		if (tooLong)
			imgnm = `${escape(name.slice(0, 30))}(&hellip;)${escape(data.ext)}`;
		return parseHTML
			`<a href="${config.MEDIA_URL}src/${data.src}"
				rel="nofollow"
				download="${fullName}"
				${tooLong && `title="${fullName}"`}
			>
				${imgnm}
			</a>`;
	}
	thumbnail(data, href) {
		const paths = this.imagePaths(),
			dims = data.dims;
		let src = paths.src + (data.src),
			thumb,
			width = dims[0],
			height = dims[1],
			thumbWidth = dims[2],
			thumbHeight = dims[3];

		// Spoilered and spoilers enabled
		if (data.spoiler && this.spoilToggle) {
			let sp = this.spoilerInfo(data);
			thumb = sp.thumb;
			thumbWidth = sp.dims[0];
			thumbHeight = sp.dims[1];
		}
		// Animated GIF thumbnails
		else if (data.ext === '.gif' && this.autoGif)
			thumb = src;
		else
			thumb = this.thumbPath(data, this.thumbStyle !== 'small');

		// Source image smaller than thumbnail and other fallbacks
		if (!thumbWidth) {
			thumbWidth = width;
			thumbHeight = height;
		}

		let linkAttrs = {
			target: '_blank',
			rel: 'nofollow',
			href: href || src
		};
		let imgAttrs = {
			src: thumb,
			width: thumbWidth,
			height: thumbHeight
		};
		// Catalog pages
		if (href) {
			// Handle the thumbnails with the HTML5 History controller
			linkAttrs.class = 'history';
			// No image hover previews
			imgAttrs.class = 'expanded';
			if(this.thumbStyle == 'hide')
				imgAttrs.style= 'display: none';
		}

		return parseHTML
			`<a ${linkAttrs}>
				<img ${imgAttrs}>
			</a>`
	}
	spoilerInfo(data) {
		let highDef = data.large || this.thumbStyle !== 'small';
		return {
			thumb: parseHTML
				`${this.imagePaths().spoil}${highDef && 's'}${data.spoiler}.png`,
			dims: config[data.large ? 'THUMB_DIMENSIONS' : 'PINKY_DIMENSIONS']
		};
	}
	postRef(num, op, desc_html) {
		let ref = '&gt;&gt;' + num;
		if (desc_html)
			ref += ' ' + desc_html;
		if (this.op && this.op != op)
			ref += ' \u27a1';
		else if (num == op && this.op == op)
			ref += ' (OP)';
		return safe(
			`<a href="${this.postURL(num, op)}" class="history">${ref}</a>`
		);
	}
	asideLink(inner, href, cls, innerCls) {
		return parseHTML
			`<aside class="act ${cls}">
				<a ${href && `href="${href}"`}
					${innerCls && ` class="${innerCls}"`}
				>
					${this.lang[inner] || inner}
				</a>
			</aside>`
	}
	replyBox() {
		return this.asideLink('reply', null, 'posting');
	}
	newThreadBox() {
		return this.asideLink('newThread', null, 'posting');
	}
}
module.exports = OneeSama;
