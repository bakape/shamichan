/*
 Renders the server-side portion of the HTML
 */

const _ = require('underscore'),
	caps = require('./caps'),
	common = require('../common/index'),
	config = require('../config'),
	db = require('../db'),
	etc = require('../util/etc'),
	lang = require('../lang/'),
	STATE = require('./state');

const RES = STATE.resources,
	actionLink = common.action_link_html,
	escape = common.escape_html,
	{parseHTML, safe} = common;

class RenderBase {
	constructor(yaku, req, resp, opts) {
		this.resp =resp;
		this.req = req;

		// Entire page, not just the contents of threads
		this.full = req.query.minimal !== 'true';
		this.lang = req.lang;
		opts.ident = req.ident;
		this.opts = opts;

		// Stores serialized post models for later stringification
		this.posts = {};
		this.initOneeSama();
		this.hidden = this.parseIntCookie('hide');

		// Top and bottom borders of the page
		yaku.once('top', nav => this.onTop(nav));
		yaku.once('bottom', () => this.onBottom());
		yaku.on('thread', post => this.onThread(post));
	}
	// Configure rendering singleton
	initOneeSama() {
		const {ident, cookies} = this.req,
			mine = this.parseIntCookie('mine'),
			links = this.links = {};
		this.oneeSama = new common.OneeSama({
			spoilToggle: cookies.spoil === 'true',
			autoGif: cookies.agif === 'true',
			eLinkify: cookies.linkify === 'true',
			lang: lang[this.lang].common,
			thumbStyle: this.req.thumbStyle,
			workMode: false,

			// Post link handler
			tamashii(num) {
				const op = db.OPs[num];
				if (op && caps.can_access_thread(ident, op)) {
					const desc = mine.has(num) && this.lang.you;
					this.callback(this.postRef(num, op, desc));
					// Pass verified post links to the client
					links[num] = op;
				}
				else
					this.callback('>>' + num);
			}
		});

		// Determine the number of [Last N] posts to diplay setting
		let lastN = cookies.lastN && parseInt(cookies.lastN, 10);
		if (!lastN || !common.reasonable_last_n(lastN))
			lastN = STATE.hot.THREAD_LAST_N;
		this.oneeSama.lastN = lastN;
	}
	// Parse list string from cookie into a set of integers
	parseIntCookie(name) {
		const ints = new Set(),
			cookie = this.req.cookies[name];
		if (cookie) {
			const split = cookie.split('/');
			for (let int of split) {
				ints.add(parseInt(int, 10));
			}
		}
		return ints;
	}
	onTop(nav) {
		// <head> and other prerendered static HTML
		let html = '';
		if (this.full)
			html += this.templateTop();

		// Subclass-specific part
		html += this.renderTop(nav);
		this.resp.write(html);
	}
	templateTop() {
		// Templates are generated two per language and cached
		const {isMobile, isRetarded} = this.req;
		this.tmpl = RES[`${isMobile ? 'mobile' : 'index'}Tmpl-${this.lang}`];
		this.tempalateIndex = 0;

		// Store time of render to prevent loading old sessions on browser
		// resume.
		let html = this.templatePart();

		// Notify the user, he/she/it should consider a brain transplant
		if (isRetarded) {
			html += '<div class="retardedBrowser">'
				+ lang[this.lang].worksBestWith + ' ';
			for (let browser of ['chrome', 'firefox', 'opera']) {
				html += `<img src="${config.MEDIA_URL}css/ui/${browser}.png">`
			}
			html += '</div>';
		}
		html += this.templatePart();

		if (!isMobile)
			html += this.imageBanner() + this.templatePart();
		return html;
	}
	// Insert the next part of the template
	templatePart() {
		return this.tmpl[this.tempalateIndex++];
	}
	imageBanner() {
		const banners = STATE.hot.BANNERS;
		if (!banners)
			return '';
		return `<img src="${config.MEDIA_URL}banners/${common.random(banners)}">`;
	}
	boardTitle() {
		const {board} = this.opts,
			title = STATE.hot.TITLES[board] || escape(board);
		this.title = title;
		return `<h1>${title}</h1>`;
	}
	/*
	 Bottom of the <threads> tag. Build backbone model skeletons
	 server-side, so there is less work to be done on the client.
	 NOTE: We could use something like rendr.js in the future.
	 */
	threadsBottom() {
		let html = parseHTML
			`<script id="postData" type="application/json">
				${JSON.stringify(_.pick(this, 'posts', 'title', 'links'))}
			</script>`;
		if (this.full)
			html += this.pageEnd();
		return html;
	}
	// <script> tags
	pageEnd() {
		let html = this.templatePart();

		// Make script loader load moderation bundle
		const {ident} = this.req;
		if (common.checkAuth('dj', ident)) {
			const keys =  JSON.stringify(_.pick(ident, 'auth', 'csrf', 'email'));
			html += `var IDENT = ${keys};`;
		}

		return html + this.templatePart();
	}
}

class Catalog extends RenderBase {
	constructor(yaku, req, resp, opts) {
		super(yaku, req, resp, opts);
		this.oneeSama.catalog = true;
	}
	renderTop() {
		// Cache so it can be resused at <threads> bottom
		const pag = this.pag
			= this.oneeSama.asideLink('return', '.', 'compact', 'history');
		return this.boardTitle() + pag + '<hr>\n<div id="catalog">';
	}
	onBottom() {
		this.resp.write('</div><hr>\n' + this.pag + this.threadsBottom());
	}
	onThread(post) {
		// Client has hidden the thread
		if (this.hidden.has(post.num))
			return;

		// Contains user-inputted data, so need to differentiate between
		// safe strings and ones needing escaping
		const html = [safe('<article>')],
			{oneeSama} = this;

		// Downscale thumbnail
		const {image} = post
		let thumb = ""

		// Account for image being deleted
		if (image) {
			const {dims} = image
			dims[2] /= 1.66
			dims[3] /= 1.66
			thumb = oneeSama.thumbnail(image, post.num)
		}

		html.push(
			safe(thumb),
			safe(parseHTML
				`<br>
				<small>
					<span title="${lang[this.lang].catalog_omit}">
						${post.replyctr}/${post.imgctr - 1}
					</span>
					${oneeSama.expansionLinks(post.num)}
				</small>
				<br>`)
		);
		if (post.subject)
			html.push(safe('<h3>「'), post.subject, safe('」</h3>'));
		html.push(oneeSama.body(post.body), safe('</article>'));
		this.resp.write(common.join(html));
	}
}
exports.Catalog = Catalog;

class Board extends RenderBase {
	constructor(yaku, req, resp, opts) {
		super(yaku, req, resp, opts);
		// XXX: Use self for now, to work around Babel.js bug when
		// es6.classes is disabled and es6.arrowFunctions is used
		const self = this;
		yaku.on('endthread', function (num) {
			self.onThreadEnd(num);
		});
		yaku.on('post', function (post) {
			self.onPost(post);
		});
	}
	renderTop(nav) {
		let html = this.boardTitle();

		// [live 0 1 2 3] [Catalog]
		const {oneeSama} = this,
			{live} = oneeSama.lang,
			cur = nav.cur_page;
		let bits = '<nav class="pagination act">';
		if (cur >= 0)
			bits += `<a href="." class="history">${live}</a>`;
		else
			bits += `<strong>${live}</strong>`;
		for (let i = 0; i != nav.pages; i++) {
			if (i != cur)
				bits += `<a href="page${i}" class="history">${i}</a>`;
			else
				bits += `<strong>${i}</strong>`;
		}
		bits += parseHTML
			`] [
			<a class="history" href="catalog">
				${oneeSama.lang.catalog}
			</a>
			</nav>`;

		this.pag = bits;
		html += bits + '<hr>\n';

		// Only render on 'live' board pages
		if (this.opts.live && !config.READ_ONLY)
			html += oneeSama.newThreadBox();
		return html;
	}
	onBottom() {
		this.resp.write(this.pag + this.threadsBottom());
	}
	onThread(post) {
		if (this.hidden.has(post.num))
			return;
		this.posts[post.num] = post;
		const {oneeSama, opts} = this,
			full = oneeSama.full = !!opts.fullPosts;
		oneeSama.op = opts.fullLinks ? false : post.num;
		let first = oneeSama.section(post, full && 'full');
		first.pop();
		this.resp.write(first.join(''));
	}
	onThreadEnd(num) {
		if (this.hidden.has(num))
			return;
		let html = '';
		if (!config.READ_ONLY)
			html += this.oneeSama.replyBox();
		html += '</section><hr>\n';
		this.resp.write(html);
	}
	onPost(post) {
		if (this.hidden.has(post.num) || this.hidden.has(post.op))
			return;
		this.posts[post.num] = post;
		this.resp.write(this.oneeSama.article(post));
	}
}
exports.Board = Board;

class Thread extends Board {
	constructor(yaku, req, resp, opts) {
		super(yaku, req, resp, opts);
	}
	renderTop() {
		let html = '';

		// Thread title
		const {board, subject, op} = this.opts;
		let title = `/${escape(board)}/ - `;
		if (subject)
			title += `${escape(subject)} (#${op})`;
		else
			title += `#${op}`;
		html += `<h1>${title}</h1>`;
		this.title = title;

		// [Bottom] [Expand Images]
		const {lang} = this.oneeSama;
		html += actionLink('#bottom', lang.bottom)
			+ '&nbsp;'
			+ actionLink('', lang.expand_images, 'expandImages')
			+ '<hr>\n';

		return html;
	}
	onBottom() {
		let {lang} = this.oneeSama;
		this.resp.write(actionLink('.', lang.return, 'bottom', 'history')
			+ '&nbsp;'
			+ actionLink('#', lang.top)
			+ `<span id="lock">${lang.locked_to_bottom}</span>`
			+ this.threadsBottom());
	}
}
exports.Thread = Thread;
