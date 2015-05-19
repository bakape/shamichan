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

var RES = STATE.resources;
var escape = common.escape_html;

function tamashii(num) {
	var op = db.OPs[num];
	if (op && caps.can_access_thread(this.ident, op))
		this.callback(this.post_ref(num, op));
	else
		this.callback('>>' + num);
}

function write_thread_html (reader, req, out, cookies, opts) {
	var oneeSama = new common.OneeSama(tamashii);
	oneeSama.tz_offset = req.tz_offset;

	opts.ident = req.ident;
	caps.augment_oneesama(oneeSama, opts);

	if (cookies.spoil == 'true')
		oneeSama.spoilToggle = true;
	if (cookies.agif == 'true')
		oneeSama.autoGif = true;
	if (cookies.rTime == 'true')
		oneeSama.rTime = true;
	if (cookies.linkify == 'true')
		oneeSama.eLinkify = true;
	if (common.thumbStyles.indexOf(cookies.thumb) >= 0)
		oneeSama.thumbStyle = cookies.thumb;
	const language = config.LANGS.indexOf(cookies.lang) > -1 ? cookies.lang
		: config.DEFAULT_LANG;
	oneeSama.lang = lang[language].common;
	var lastN = cookies.lastn && parseInt(cookies.lastn, 10);
	if (!lastN || !common.reasonable_last_n(lastN))
		lastN = STATE.hot.THREAD_LAST_N;
	oneeSama.lastN = lastN;

	var hidden = {};
	if (cookies.hide && !caps.can_moderate(req.ident)) {
		cookies.hide.slice(0, 200).split(',').forEach(function (num) {
			num = parseInt(num, 10);
			if (num)
				hidden[num] = null;
		});
	}

	/*
	 Build backbone model skeletons server-side, so there is less work to be done
	 on the client.
	 NOTE: We could use soemthing like rendr.js in the future.
	 */
	var posts = {};
	const readOnly = config.READ_ONLY
			|| config.READ_ONLY_BOARDS.indexOf(opts.board) >= 0;
	// Cache pagination, as not to render twice
	var pag;
	// Top and bottom borders of the <threads> tag
	reader.once('top', function(nav) {
		// Navigation info is used to build pagination. None on thread pages
		if (!nav)
			out.write(threadsTop(oneeSama));
		else {
			pag = pagination(nav, oneeSama);
			out.write(pag);
		}
		out.write('<hr>\n');
		// Only render on 'live' board pages
		if (nav && !readOnly && !/\/page\d+/.test(req.url))
			out.write(oneeSama.newThreadBox());
	});
	reader.once('bottom', function() {
		// Serialze post collection and add as inlined JSON
		out.write(common.parseHTML
			`<script id="postData" type="application/json">
				${JSON.stringify(posts)}
			</script>`
		);
		out.write(pag || threadsBottom(oneeSama));
	});

	reader.on('thread', function (op_post, omit) {
		if (op_post.num in hidden)
			return;
		op_post.omit = omit || 0;
		// Currently only calculated client-side
		op_post.image_omit = 0;
		op_post.replies = [];
		posts[op_post.num] = op_post;

		const full = oneeSama.full = !!opts.fullPosts;
		oneeSama.op = opts.fullLinks ? false : op_post.num;
		var first = oneeSama.monomono(op_post, full && 'full');
		first.pop();
		out.write(first.join(''));

		reader.once('endthread', function() {
			if (!readOnly)
				out.write(oneeSama.replyBox());
			out.write('</section><hr>\n');
		});
	});

	reader.on('post', function (post) {
		if (post.num in hidden || post.op in hidden)
			return;
		posts[post.num] = post;
		// Add to parent threads replies
		posts[post.op].replies.push(post.num);
		out.write(oneeSama.mono(post));
	});
}
exports.write_thread_html = write_thread_html;

// [live 0 1 2 3] [Catalog]
function pagination(info, oneeSama) {
	const live = oneeSama.lang.live,
		cur = info.cur_page;
	let bits = '<nav class="pagination act">';
	if (cur >= 0)
		bits += `<a href="." class="history">${live}</a>`;
	else
		bits += `<strong>${live}</strong>`;
	let start = 0,
		end = info.pages,
		step = 1;
	if (info.ascending) {
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
	return bits;
}


function threadsTop(oneeSama) {
	return common.action_link_html('#bottom', oneeSama.lang.bottom)
		+ '&nbsp;'
		+ common.action_link_html(
			'',
			oneeSama.lang.expand_images,
			'expandImages'
		);
}

function threadsBottom(oneeSama) {
	return common.action_link_html('.',	oneeSama.lang.return, 'bottom', 'history')
		+ '&nbsp;'
		+ common.action_link_html('#', oneeSama.lang.top)
		+ common.parseHTML
			`<span id="lock" style="visibility: hidden;">
				${oneeSama.lang.locked_to_bottom}
			</span>`;
}

function make_link_rels(board, bits) {
	var path = config.MEDIA_URL + 'css/',
		// Object of CSS versions
		css = STATE.hot.css;

	bits.push(['stylesheet', path + css['base.css']]);

	var theme_css = css[STATE.hot.BOARD_CSS[board] + '.css'];
	bits.push(['stylesheet', path + theme_css, 'theme']);

	return bits.map(function (p) {
		var html = '\t<link rel="'+p[0]+'" href="'+p[1]+'"';
		if (p[2])
			html += ' id="' + p[2] + '"';
		return html + '>\n';
	}).join('');
}

function write_board_head (out, board, nav, language) {
	var indexTmpl = RES['indexTmpl-' + language];
	var title = STATE.hot.TITLES[board] || escape(board);
	var metaDesc = "Real-time imageboard";

	var i = 0;
	out.write(indexTmpl[i++]);
	out.write(title);
	out.write(indexTmpl[i++]);
	out.write(escape(metaDesc));
	out.write(indexTmpl[i++]);
	out.write(make_board_meta(board, nav));
	out.write(indexTmpl[i++]);
	out.write(indexTmpl[i++]);
	out.write(imageBanner());
	out.write(title);
	out.write(indexTmpl[i++]);
}
exports.write_board_head = write_board_head;

function imageBanner() {
	var b = config.BANNERS;
	if (!b)
		return '';
	return `<img id="imgBanner" src="${config.MEDIA_URL}banners/`
		+ b[Math.floor(Math.random() * b.length)] + '"><br>';
}

function write_board_title(out, board){
	var title = STATE.hot.TITLES[board] || escape(board);
	out.write(`<h1>${imageBanner()}${title}</h1>`);
}
exports.write_board_title = write_board_title;

function write_thread_head (out, board, op, opts) {
	const indexTmpl = RES['indexTmpl-' + opts.lang];
	var title = '/'+escape(board)+'/';
	if (opts.subject)
		title += ' - ' + escape(opts.subject) + ' (#' + op + ')';
	else
		title += ' - #' + op;
	var metaDesc = "Real-time imageboard thread";

	var i = 0;
	out.write(indexTmpl[i++]);
	out.write(title);
	out.write(indexTmpl[i++]);
	out.write(escape(metaDesc));
	out.write(indexTmpl[i++]);
	out.write(make_thread_meta(board, op, opts.abbrev));
	out.write(indexTmpl[i++]);
	out.write(indexTmpl[i++]);
	out.write(imageBanner());
	out.write(title);
	out.write(indexTmpl[i++]);
}
exports.write_thread_head = write_thread_head;

function write_thread_title(out, board, op, opts){
	var title = '/'+escape(board)+'/';
	if (opts.subject)
		title += ' - ' + escape(opts.subject) + ' (#' + op + ')';
	else
		title += ' - #' + op;
	out.write(`<h1>${imageBanner()}${title}</h1>`);
}
exports.write_thread_title = write_thread_title;

function make_board_meta(board, info) {
	var bits = [];
	if (info.cur_page >= 0)
		bits.push(['index', '.']);
	return make_link_rels(board, bits);
}

function make_thread_meta(board, num, abbrev) {
	var bits = [['index', '.']];
	if (abbrev)
		bits.push(['canonical', num]);
	return make_link_rels(board, bits);
}

function write_page_end (out, ident, language) {
	const tmpl = 'indexTmpl-' + language;
	out.write(RES[tmpl][RES[tmpl].length - 1]);
	if (ident) {
		if (caps.can_administrate(ident))
			out.write('<script src="../admin.js"></script>\n');
		else if (caps.can_moderate(ident))
			out.write('<script src="../mod.js"></script>\n');
	}
}
exports.write_page_end = write_page_end;
