/*
 Renders the server-side portion of the HTML
 */
'use strict';

var caps = require('./caps'),
	common = require('../common/index'),
	config = require('../config'),
	db = require('../db'),
	lang = require('../lang/'),
	STATE = require('./state'),
	web = require('./web');

let RES = STATE.resources,
	escape = common.escape_html;

class Render {
	constructor(yaku, req, resp, opts) {
		this.resp =resp;
		this.req = req;
		this.parseRequest();
		this.tmpl = RES['indexTmpl-' + this.lang];
		this.readOnly = config.READ_ONLY
			|| config.READ_ONLY_BOARDS.indexOf(opts.board) >= 0;
		opts.ident = req.ident;
		this.opts = opts;
		this.posts = {};
		this.initOneeSama().getHidden();

		// Top and bottom borders of the page
		yaku.once('top', this.onTop.bind(this));
		yaku.once('bottom', this.onBottom.bind(this));

		yaku.on('thread', this.onThread.bind(this));
		yaku.on('endthread', this.onThreadEnd.bind(this));
		yaku.on('post', this.onPost.bind(this));
	}

	parseRequest() {
		let req = this.req;
		// Entire page, not just the contents of threads
		this.full = req.query.minimal !== 'true';
		const cookies = this.cookies = web.parse_cookie(req.headers.cookie);
		this.lang = config.LANGS.indexOf(cookies.lang) > -1 ? cookies.lang
			: config.DEFAULT_LANG;
	}

	// Read cookies and configure rendering singleton
	initOneeSama() {
		const ident = this.req.ident;
		let oneeSama = new common.OneeSama(function(num) {
			// Post link handler
			const op = db.OPs[num];
			if (op && caps.can_access_thread(ident, op))
				this.callback(this.post_ref(num, op));
			else
				this.callback('>>' + num);
		});
		let cookies = this.cookies;
		oneeSama.tz_offset = this.req.tz_offset;
		caps.augment_oneesama(oneeSama, this.opts);

		if (cookies.spoil === 'true')
			oneeSama.spoilToggle = true;
		if (cookies.agif === 'true')
			oneeSama.autoGif = true;
		if (cookies.rTime === 'true')
			oneeSama.rTime = true;
		if (cookies.linkify === 'true')
			oneeSama.eLinkify = true;
		if (common.thumbStyles.indexOf(cookies.thumb) >= 0)
			oneeSama.thumbStyle = cookies.thumb;
		oneeSama.lang = lang[this.lang].common;
		let lastN = cookies.lastn && parseInt(cookies.lastn, 10);
		if (!lastN || !common.reasonable_last_n(lastN))
			lastN = STATE.hot.THREAD_LAST_N;
		oneeSama.lastN = lastN;
		this.oneeSama = oneeSama;
		return this;
	}

	// Read hidden posts from cookie
	getHidden() {
		let hidden = new Set();
		const hide = this.cookies.hide;
		if (hide && !caps.can_moderate(this.req.ident)) {
			const toHide = hide.slice(0, 200).split(',');
			for (let i = 0, l = toHide.length; i < l; i++) {
				const num = parseInt(toHide[i], 10);
				if (num)
					hidden.add(num);
			}
		}
		this.hidden = hidden;
	}

	// Top of the page
	onTop(nav) {
		let resp = this.resp;
		const opts = this.opts;
		// <head> and other prerendered static HTML
		if (this.full)
			resp.write(this.tmpl[0]);
		const isThread = opts.isThread;
		if (isThread)
			this.threadTitle().threadTop();
		else
			this.boardTitle().pagination(nav);
		resp.write('<hr>\n');
		// Only render on 'live' board pages
		if (!isThread && !this.readOnly && !/\/page\d+/.test(this.req.url))
			resp.write(this.oneeSama.newThreadBox());
	}

	onBottom() {
		let resp = this.resp;
		resp.write(this.pag || this.threadBottom());

		/*
		 Build backbone model skeletons server-side, so there is less work to be
		 done on the client.
		 NOTE: We could use something like rendr.js in the future.
		 */
		resp.write(common.parseHTML
			`<script id="postData" type="application/json">
				${JSON.stringify({
					posts: this.posts,
					title: this.title
				})}
			</script>`
		);
		if (this.full)
			this.pageEnd();
	}

	onThread(post, omit) {
		if (this.hidden.has(post.num))
			return;
		post.omit = omit || 0;
		// Currently only calculated client-side
		post.image_omit = 0;
		post.replies = [];
		this.posts[post.num] = post;

		let oneeSama = this.oneeSama;
		const opts = this.opts,
			full = oneeSama.full = !!opts.fullPosts;
		oneeSama.op = opts.fullLinks ? false : post.num;
		let first = oneeSama.monomono(post, full && 'full');
		first.pop();
		this.resp.write(first.join(''));
	}

	onThreadEnd() {
		let resp = this.resp;
		if (!this.readOnly)
			resp.write(this.oneeSama.replyBox());
		resp.write('</section><hr>\n');
	}

	onPost(post) {
		const hidden = this.hidden;
		if (hidden.has(post.num) || hidden.has(post.op))
			return;
		let posts = this.posts;
		posts[post.num] = post;
		// Add to parent threads replies
		posts[post.op].replies.push(post.num);
		this.resp.write(this.oneeSama.mono(post));
	}

	threadTitle() {
		let title = `/${escape(this.opts.board)}/ - `;
		const subject = this.opts.subject,
			op = this.opts.op;
		if (subject)
			title += `${escape(subject)} (#${op})`;
		else
			title += `#${op}`;
		this.resp.write(`<h1>${this.imageBanner()}${title}</h1>`);
		this.title = title;
		return this;
	}

	boardTitle() {
		const board = this.opts.board,
			title = STATE.hot.TITLES[board] || escape(board);
		this.resp.write(`<h1>${this.imageBanner()}${title}</h1>`);
		this.title = title;
		return this;
	}

	imageBanner() {
		const banners = config.BANNERS;
		if (!banners)
			return '';
		return common.parseHTML
			`<img id="imgBanner"
				src="${config.MEDIA_URL}banners/${common.random(banners)}"
			>
			<br>`;
	}

	// Top of a <threads> element on a thread page
	threadTop() {
		this.resp.write(
			common.action_link_html('#bottom', this.oneeSama.lang.bottom)
			+ '&nbsp;'
			+ common.action_link_html(
				'',
				this.oneeSama.lang.expand_images,
				'expandImages'
			)
		);
	}

	// [live 0 1 2 3] [Catalog]
	pagination(nav) {
		let oneeSama = this.oneeSama;
		const live = oneeSama.lang.live,
			cur = nav.cur_page;
		let bits = '<nav class="pagination act">';
		if (cur >= 0)
			bits += `<a href="." class="history">${live}</a>`;
		else
			bits += `<strong>${live}</strong>`;
		let start = 0,
			end = nav.pages,
			step = 1;
		if (nav.ascending) {
			start = end - 1;
			end = step = -1;
		}
		for (let i = start; i != end; i += step) {
			if (i != cur)
				bits += `<a href="page${i}" class="history">${i}</a>`;
			else
				bits += `<strong>${i}</strong>`;
		}
		bits += `] [<a class="catalogLink">${oneeSama.lang.catalog}</a></nav>`;
		this.resp.write(bits);
		this.pag = bits;
	}

	// Bottom of the <threads> tag on thread pages
	threadBottom() {
		let oneeSama = this.oneeSama;
		return common.action_link_html('.', oneeSama.lang.return, 'bottom',
				'history')
			+ '&nbsp;'
			+ common.action_link_html('#', oneeSama.lang.top)
			+ common.parseHTML
				`<span id="lock" style="visibility: hidden;">
					${oneeSama.lang.locked_to_bottom}
				</span>`;
	}

	// <script> tags
	pageEnd() {
		let resp = this.resp;
		resp.write(this.tmpl[1]);
		const ident = this.req.ident;
		if (ident) {
			if (caps.can_administrate(ident))
				resp.write('<script src="../admin.js"></script>\n');
			else if (caps.can_moderate(ident))
				resp.write('<script src="../mod.js"></script>\n');
		}
	}
}
module.exports = Render;
