var caps = require('./caps'),
    common = require('../common'),
    config = require('../config'),
    db = require('../db'),
    imager = require('../imager'),
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

exports.write_thread_html = function (reader, req, out, opts) {
	var oneeSama = new common.OneeSama(tamashii);

	opts.ident = req.ident;
	caps.augment_oneesama(oneeSama, opts);

	var cookies = web.parse_cookie(req.headers.cookie);
	if (common.thumbStyles.indexOf(cookies.thumb) >= 0)
		oneeSama.thumbStyle = cookies.thumb;

	var lastN = cookies.lastn && parseInt(cookies.lastn, 10);
	if (!lastN || !common.reasonable_last_n(lastN))
		lastN = config.THREAD_LAST_N;
	oneeSama.lastN = lastN;

	var hidden = {};
	if (cookies.hide && !caps.can_moderate(req.ident)) {
		cookies.hide.slice(0, 200).split(',').forEach(function (num) {
			num = parseInt(num, 10);
			if (num)
				hidden[num] = null;
		});
	}

	reader.on('thread', function (op_post, omit, image_omit) {
		if (op_post.num in hidden)
			return;
		op_post.omit = omit;
		var full = oneeSama.full = !!opts.fullPosts;
		oneeSama.op = opts.fullLinks ? false : op_post.num;
		var first = oneeSama.monomono(op_post, full && 'full');
		first.pop();
		out.write(first.join(''));
		if (omit) {
			var o = common.abbrev_msg(omit, image_omit);
			if (opts.loadAllPostsLink)
				o += ' '+common.action_link_html(op_post.num,
						'See all');
			out.write('\t<span class="omit">'+o+'</span>\n');
		}
		reader.once('endthread', close_section);
	});
	reader.on('post', function (post) {
		if (post.num in hidden || post.op in hidden)
			return;
		out.write(oneeSama.mono(post));
	});

	function close_section() {
		out.write('</section><hr>\n');
	}
};

function make_link_rels(board, bits) {
	var path = imager.config.MEDIA_URL + 'css/';
	bits.push(['stylesheet', path + STATE.hot.BASE_CSS]);
	bits.push(['stylesheet', path + STATE.hot.BOARD_CSS[board], 'theme']);
	bits.push(['stylesheet', path + 'gravitas-v1.css']);
	return bits.map(function (p) {
		var html = '\t<link rel="'+p[0]+'" href="'+p[1]+'"';
		if (p[2])
			html += ' id="' + p[2] + '"';
		return html + '>\n';
	}).join('');
}

exports.write_board_head = function (out, board, nav) {
	var indexTmpl = RES.indexTmpl;
	var title = STATE.hot.TITLES[board] || escape(board);
	out.write(indexTmpl[0]);
	out.write(title);
	out.write(indexTmpl[1]);
	out.write(make_board_meta(board, nav));
	out.write(indexTmpl[2]);
	if (RES.navigationHtml)
		out.write(RES.navigationHtml);
	out.write(indexTmpl[3]);
	out.write(title);
	out.write(indexTmpl[4]);
};

exports.write_thread_head = function (out, board, op, subject, abbrev) {
	var indexTmpl = RES.indexTmpl;
	var title = '/'+escape(board)+'/ - ';
	if (subject)
		title += escape(subject) + ' (#' + op + ')';
	else
		title += '#' + op;

	out.write(indexTmpl[0]);
	out.write(title);
	out.write(indexTmpl[1]);
	out.write(make_thread_meta(board, op, abbrev));
	out.write(indexTmpl[2]);
	if (RES.navigationHtml)
		out.write(RES.navigationHtml);
	out.write(indexTmpl[3]);
	out.write('Thread #' + op);
	out.write(indexTmpl[4]);
	out.write(common.action_link_html('#bottom', 'Bottom'));
	out.write('<hr>\n');
};

function make_board_meta(board, info) {
	var bits = [];
	if (info.cur_page >= 0)
		bits.push(['index', '.']);
	if (info.prev_page)
		bits.push(['prev', info.prev_page]);
	if (info.next_page)
		bits.push(['next', info.next_page]);
	return make_link_rels(board, bits);
}

function make_thread_meta(board, num, abbrev) {
	var bits = [['index', '.']];
	if (abbrev)
		bits.push(['canonical', num]);
	return make_link_rels(board, bits);
}

exports.make_pagination_html = function (info) {
	var bits = ['<nav class="pagination">'], cur = info.cur_page;
	if (cur >= 0)
		bits.push('<a href=".">live</a>');
	else
		bits.push('<strong>live</strong>');
	var start = 0, end = info.pages, step = 1;
	if (info.ascending) {
		start = end - 1;
		end = step = -1;
	}
	for (var i = start; i != end; i += step) {
		if (i != cur)
			bits.push('<a href="page' + i + '">' + i + '</a>');
		else
			bits.push('<strong>' + i + '</strong>');
	}
	if (info.next_page)
		bits.push(' <input type="button" value="Next">');
	bits.push('</nav>');
	return bits.join('');
};

var returnHTML = common.action_link_html('.', 'Return').replace(
		'span', 'span id="bottom"');

exports.write_page_end = function (out, ident, returnLink) {
	out.write(RES.indexTmpl[5]);
	if (returnLink)
		out.write(returnHTML);
	else if (RES.navigationHtml)
		out.write('<br><br>' + RES.navigationHtml);
	out.write(RES.indexTmpl[6]);
	if (ident) {
		if (caps.can_administrate(ident))
			out.write('<script src="../admin.js"></script>\n');
		else if (caps.can_moderate(ident))
			out.write('<script src="../mod.js"></script>\n');
	}
};
