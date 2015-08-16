/*
 Server the HTML part of pages
 */

let _ = require('underscore'),
	caps = require('../caps'),
	common = require('../../common'),
	config = require('../../config'),
	db = require('../../db'),
	express = require('express'),
	hooks = require('../../util/hooks'),
	Render = require('../render'),
	state = require('../state'),
	util = require('./util'),
	winston = require('winston');

let router = module.exports = express.Router(),
	RES = state.resources;

const vanillaHeaders = {
	'Content-Type': 'text/html; charset=UTF-8',
	'X-Frame-Options': 'sameorigin',
	'Cache-Control': `max-age=0, must-revalidate, private`
};

router.get('/', function(req, res) {
	res.redirect(`/${config.DEFAULT_BOARD}/`)
});

// Redirect `/board` to `/board/` The client parses the URL to determine
// what page it is on. So we need the trailing slash for easier board
// determination.
router.get(/^\/(\w+)$/, function(req, res) {
	res.redirect(`/${req.params[0]}/`);
});

// /board/ and /board/catalog pages
router.get(/^\/(\w+)\/(catalog)?$/,
	util.boardAccess,
	function(req, res, next) {
		const board = req.board;
		let yaku = res.yaku = new db.Yakusoku(board, req.ident);
		const catalog = !!req.params[1];
		yaku.get_tag(catalog ? -2 : -1);
		yaku.once('begin', function (thread_count, post_count) {
			// More efficient to confirm we actually need to retrieve and render
			// the page before fully creating the Reader() -> Render() ->
			// response pipeline
			if (!buildEtag(req, res, post_count))
				return yaku.disconnect();
			res.opts = {
				board,
				catalog,
				thread_count,
				post_count
			};
			next();
		});
	},
	function(req, res, next) {
		const opts = res.opts,
			board = opts.board;
		let yaku = res.yaku;
		new Render(yaku, req, res, {
			fullLinks: true,
			board,
			isThread: false,
			live: true,
			catalog: opts.catalog
		});
		yaku.emit('top', page_nav(opts.thread_count, -1));
		yaku.once('error', function(err) {
			winston.error('index:' + err);
			next();
		});
		yaku.once('end', function() {
			yaku.emit('bottom');
			next();
		});
	},
	finish
);

router.get(/^\/(\w+)\/page(\d+)$/,
	util.boardAccess,
	function(req, res, next) {
		const board = req.board,
			page = parseInt(req.params[1], 10);
		let yaku = new db.Yakusoku(board, req.ident);
		yaku.get_tag(page);

		// The page might be gone, becaue a thread was deleted
		yaku.once('nomatch', function() {
			res.redirect('.');
			yaku.disconnect();
		});
		yaku.once('begin', function(threadCount, postCount) {
			if (!buildEtag(req, res, postCount))
				return yaku.disconnect();
			res.yaku = yaku;
			res.opts = {
				board,
				page,
				threadCount
			};
			next();
		});
	},
	function(req, res, next) {
		const opts = res.opts,
			board = opts.board,
			page = opts.page;
		let yaku = res.yaku;

		new Render(yaku, req, res, {
			fullLinks: true,
			board,
			isThread: false
		});
		yaku.emit('top',
			page_nav(opts.threadCount, page)
		);
		yaku.once('end', function() {
			yaku.emit('bottom');
			next();
		});
		yaku.once('error', function(err) {
			winston.error(`page${page}: ${err}`);
			next();
		});
	},
	finish
);

// Thread pages
router.get(/^\/(\w+)\/(\d+)/,
	util.boardAccess,
	function(req, res, next) {
		const board = req.board,
			num = parseInt(req.params[1], 10),
			ident = req.ident;
		if (!num)
			return res.sendStatus(404);

		// We need to validate that the requested post number, is in fact a
		// thread and not a reply
		const op = db.OPs[num];
		if (!op)
			return res.sendStatus(404);
		if (!db.OP_has_tag(board, op)) {
			let tag = db.first_tag_of(op);
			if (tag) {
				if (!caps.can_access_board(ident, tag))
					return res.sendStatus(404);
				return redirect_thread(res, num, op, tag, req.url);
			}
			else {
				winston.warn(`Orphaned post ${num} with tagless OP ${op}`);
				return res.sendStatus(404);
			}
		}
		if (op != num)
			return redirect_thread(res, num, op);

		if (!caps.can_access_thread(ident, op))
			return res.sendStatus(404);

		let yaku = new db.Yakusoku(board, ident),
			reader = new db.Reader(ident),
			opts = {redirect: true};

		const lastN = detect_last_n(req.query);
		if (lastN)
			opts.abbrev = lastN + state.hot.ABBREVIATED_REPLIES;
		
		reader.get_thread(board, num, opts);
		reader.once('nomatch', function() {
			res.sendStatus(404);
			yaku.disconnect();
		});
		reader.once('redirect', function(op) {
			redirect_thread(res, num, op);
			yaku.disconnect();
		});
		reader.once('begin', function(preThread) {
			let extra = '';
			if (lastN)
				extra += '-last' + lastN;
			if (preThread.locked)
				extra += '-locked';
			if (!buildEtag(req, res, preThread.hctr, extra))
				return yaku.disconnect();

			res.yaku = yaku;
			res.reader = reader;
			res.opts = {
				board,
				op,
				subject: preThread.subject,
				abbrev: opts.abbrev
			};
			next();
		});
	},
	function(req, res, next) {
		const opts = res.opts;
		let reader = res.reader;
		new Render(reader, req, res, {
			fullPosts: true,
			board: opts.board,
			op: opts.op,
			subject: opts.subject,
			isThread: true
		});
		reader.emit('top');
		reader.once('end', function() {
			reader.emit('bottom');
			next();
		});
		reader.once('error', on_err);
		res.yaku.once('error', on_err);

		function on_err(err) {
			winston.error(`thread ${num}:`, err);
			next();
		}
	},
	finish
);

// Build an eTag in accordance to the board/thread progress counter and
// cookie parameters, as those effect the HTML
function buildEtag(req, res, ctr, extra) {
	let etag = parseCookies(req, ctr);
	if (config.DEBUG) {
		res.set(util.noCacheHeaders);
		return true;
	}
	if (extra)
		etag += extra;
	if (!!req.ident.auth)
		etag += '-auth';

	// etags match. No need to rerender.
	if (req.headers['If-None-Match'] === etag) {
		res.sendStatus(304);
		return false;
	}

	let headers = _.clone(vanillaHeaders);
	headers.ETag = etag;
	res.set(headers);

	return true;
}

function parseCookies(req, ctr) {
	const cookies = req.cookies,
		lang = req.lang = ~config.LANGS.indexOf(cookies.lang)
			? cookies.lang : config.DEFAULT_LANG;

	let etag = `W/${ctr}-${RES['indexHash-' + lang]}-${lang}`;

	// Attach thumbnail mode to etag
	const thumb = cookies.thumb,
		styles = common.thumbStyles;
	etag += '-' + (~styles.indexOf(thumb) ? thumb : styles[0]);

	const etags = ['spoil', 'agif', 'rtime', 'linkify'];
	for (let tag of etags) {
		if (tag in cookies)
			etag += `-${tag}:${cookies[tag]}`;
	}

	return etag;
}

function finish(req, res) {
	res.yaku.disconnect();
	res.end();
}

// Pack page navigation data in an object for easier passing downstream
function page_nav(threads, cur_page) {
	return {
		pages: Math.max(Math.ceil(threads / state.hot.THREADS_PER_PAGE)),
		threads,
		cur_page
	};
}

// Redirects '/board/num', when num point to a reply, not a thread
function redirect_thread(res, num, op, tag, url) {
	let path = tag ? `../${tag}/${op}` : `./${op}`;

	// Reapply query strings, so we don't screw up the History API by
	// retrieving a full page
	const query = url && url.split('?')[1];
	if (query)
		path += '?' + query;
	path += '#' + num;
	
	res.redirect(path);
}

function detect_last_n(query) {
	if (query.last) {
		const n = parseInt(query.last, 10);
		if (common.reasonable_last_n(n))
			return n;
	}
	return 0;
}
