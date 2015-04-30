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

/*
 * XXX: This entire module is a mess of redundancy and repitition.
 * Everything but write_thread_html should be moved to OneeSama or the static
 * templates. Some things can be offloaded to the client.
 */

exports.write_thread_html = function (reader, req, out, cookies, opts) {
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

	// Top and bottom borders of the <threads> tag
	// Chache pagination, as not to render twice
	var notReadOnly = !config.READ_ONLY
			&& config.READ_ONLY_BOARDS.indexOf(opts.board) < 1,
		pag;
	reader.once('top', function(nav) {
		// Navigation info is used to build pagination. None on thread pages
		if (!nav)
			out.write(threadsTop(oneeSama));
		else {
			pag = pagination(nav, oneeSama);
			out.write(pag);
		}
		out.write('<hr class="sectionHr">\n');
		if (nav && notReadOnly)
			out.write(oneeSama.newThreadBox());
	});
	reader.once('bottom', function() {
		out.write(pag || threadsBottom(oneeSama));
	});

	var write_see_all_link;

	reader.on('thread', function (op_post, omit, image_omit) {
		if (op_post.num in hidden)
			return;
		op_post.omit = omit;
		var full = oneeSama.full = !!opts.fullPosts;
		oneeSama.op = opts.fullLinks ? false : op_post.num;
		var first = oneeSama.monomono(op_post, full && 'full');
		first.pop();
		out.write(first.join(''));

		write_see_all_link = omit && function (first_reply_num) {
			var o = oneeSama.lang.abbrev_msg(omit, image_omit);
			if (opts.loadAllPostsLink) {
				var url = '' + op_post.num;
				if (first_reply_num)
					url += '#' + first_reply_num;
				o += ' '+common.action_link_html(url, oneeSama.lang.see_all);
			}
			out.write('\t<span class="omit">'+o+'</span>\n');
		};

		reader.once('endthread', function() {
			if (notReadOnly)
				out.write(oneeSama.replyBox());
			out.write('</section><hr class="sectionHr">\n');
		});
	});

	reader.on('post', function (post) {
		if (post.num in hidden || post.op in hidden)
			return;
		if (write_see_all_link) {
			write_see_all_link(post.num);
			write_see_all_link = null;
		}
		out.write(oneeSama.mono(post));
	});
};

// [live 0 1 2 3] [Catalog]
function pagination(info, oneeSama) {
	const live = oneeSama.lang.live,
		cur = info.cur_page;
	var bits = '<nav class="pagination act">';
	if (cur >= 0)
		bits += `<a href="." class="history">${live}</a>`;
	else
		bits += `<strong>${live}</strong>`;
	var start = 0, end = info.pages, step = 1;
	if (info.ascending) {
		start = end - 1;
		end = step = -1;
	}
	for (var i = start; i != end; i += step) {
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
		+ common.action_link_html('#', oneeSama.lang.top);
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

exports.write_board_head = function (out, board, nav, language) {
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
};

function imageBanner() {
	var b = config.BANNERS;
	if (!b)
		return '';
	return `<img id="imgBanner" src="${config.MEDIA_URL}banners/`
		+ b[Math.floor(Math.random() * b.length)] + '"><br>';
}

exports.write_board_title = function(out, board){
	var title = STATE.hot.TITLES[board] || escape(board);
	out.write(`<h1>${imageBanner()}${title}</h1>`);
};

exports.write_thread_head = function (out, board, op, opts) {
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
};

exports.write_thread_title = function(out, board, op, opts){
	var title = '/'+escape(board)+'/';
	if (opts.subject)
		title += ' - ' + escape(opts.subject) + ' (#' + op + ')';
	else
		title += ' - #' + op;
	out.write(`<h1>${imageBanner()}${title}</h1>`);
};

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

exports.write_page_end = function (out, ident, language) {
	const tmpl = 'indexTmpl-' + language;
	out.write(RES[tmpl][RES[tmpl].length - 1]);
	if (ident) {
		if (caps.can_administrate(ident))
			out.write('<script src="../admin.js"></script>\n');
		else if (caps.can_moderate(ident))
			out.write('<script src="../mod.js"></script>\n');
	}
};
