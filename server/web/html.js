/*
 Serve the HTML part of pages
 */

const _ = require('underscore'),
	caps = require('../caps'),
	common = require('../../common'),
	config = require('../../config'),
	db = require('../../db'),
	etc = require('../../util/etc'),
	express = require('express'),
	hooks = require('../../util/hooks'),
	render = require('../render'),
	state = require('../state'),
	util = require('./util'),
	uaParser = require('ua-parser-js'),
	winston = require('winston');

const router = module.exports = express.Router(),
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
		const yaku = res.yaku = new db.Yakusoku(req.board, req.ident),
			catalog = !!req.params[1];
		yaku.get_tag(catalog ? -2 : -1);
		yaku.once('begin', function (thread_count, post_count) {
			// More efficient to confirm we actually need to retrieve and render
			// the page before fully creating the Reader() -> ../render ->
			// response pipeline
			if (!buildEtag(req, res, post_count))
				return yaku.disconnect();
			res.opts = {
				catalog,
				thread_count
			};
			next();
		});
	},
	function(req, res, next) {
		const {opts, yaku} = res;
		new render[opts.catalog ? 'Catalog' : 'Board'](yaku, req, res, {
			fullLinks: true,
			board: req.board,
			live: true
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
		const {board} = req,
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
				page,
				threadCount
			};
			next();
		});
	},
	function(req, res, next) {
		const {opts, yaku} = res,
			{page, threadCount} = opts;
		new render.Board(yaku, req, res, {
			fullLinks: true,
			board: req.board
		});
		yaku.emit('top', page_nav(threadCount, page));
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
		const {board} = req,
			num = parseInt(req.params[1], 10),
			{ident} = req;
		if (!num)
			return res.sendStatus(404);
		
		// Validate the post number points to a thread. If not, validate it
		// points to a reply and redirect.
		if (!db.boards[num]) {
			const actualOP = db.OPs[num];
			if (actualOP)
				return redirect_thread(res, num, actualOP, board, req.url);
			return res.sendStatus(404);
		}

		if (!caps.can_access_thread(ident, num))
			return res.sendStatus(404);

		const yaku = new db.Yakusoku(board, ident),
			reader = new db.Reader(ident),
			opts = {};

		const lastN = detect_last_n(req.query);
		if (lastN)
			opts.abbrev = lastN + state.hot.ABBREVIATED_REPLIES;
		
		reader.get_thread(num, opts);
		reader.once('nomatch', function() {
			res.sendStatus(404);
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
				op: num,
				subject: preThread.subject,
				abbrev: opts.abbrev
			};
			next();
		});
	},
	function(req, res, next) {
		const {opts, reader, yaku} = res;
		new render.Thread(reader, req, res, {
			fullPosts: true,
			board: opts.board,
			op: opts.op,
			subject: opts.subject
		});
		reader.emit('top');
		reader.once('end', function() {
			reader.emit('bottom');
			next();
		});
		reader.once('error', on_err);
		yaku.once('error', on_err);

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

	// Parse UserAgent
	const parsed = uaParser(req.header('user-agent'));
	if (parsed.device.type) {
		req.isMobile = true;
		etag += '-mobile';
	}
	if (['Chrome', 'Firefox', 'Opera'].indexOf(parsed.browser.name) < 0 )
		req.isRetarded = true;

	if (config.DEBUG) {
		res.set(util.noCacheHeaders);
		return true;
	}
	if (extra)
		etag += extra;
	if (req.ident.auth)
		etag += '-auth';

	// etags match. No need to rerender.
	if (req.headers['If-None-Match'] === etag) {
		res.sendStatus(304);
		return false;
	}

	const headers = _.clone(vanillaHeaders);
	headers.ETag = etag;
	res.set(headers);

	return true;
}

function parseCookies(req, ctr) {
	const {cookies} = req,
		lang = req.lang =  etc.resolveConfig(config.LANGS, cookies.lang,
			config.DEFAULT_LANG);

	let etag = `W/${ctr}-${RES['indexHash-' + lang]}-${lang}`;

	// Attach thumbnail mode to etag
	const styles = common.thumbStyles,
		style = etc.resolveConfig(styles, cookies.thumb, styles[0]);
	req.thumbStyle = style;
	etag += '-' + style;

	for (let tag of ['spoil', 'agif', 'linkify']) {
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
function redirect_thread(res, num, op, board, url) {
	let path = board ? `../${board}/${op}` : `./${op}`;

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
