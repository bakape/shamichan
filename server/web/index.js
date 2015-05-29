/*
 Webserver
 */

'use strict';

let _ = require('underscore'),
	caps = require('../caps'),
	cookieParser = require('cookie-parser'),
	common = require('../../common'),
	compress = require('compression'),
	config = require('../../config'),
	db = require('../../db'),
	express = require('express'),
	http = require('http'),
	imager = require('../../imager/daemon'),
	Render = require('../render'),
	state = require('../state'),
	util = require('./util'),
	websocket = require('./websocket');

let app = express(),
	server = http.createServer(app);
websocket.start(server);
app.enable('strict routing');
server.listen(config.LISTEN_PORT);

// Image uploads
app.post('/upload/', imager.new_upload);

if (config.GZIP)
	app.use(compress());
if (config.SERVE_STATIC_FILES)
	app.use(express.static('www'));
app.use(cookieParser());

// Authentication
app.use(function(req, res, next) {
	let ip = req.connection.remoteAddress;
	if (config.TRUST_X_FORWARDED_FOR)
		ip = util.parse_forwarded_for(req.headers['x-forwarded-for']) || ip;
	if (!ip) {
		res.set({'Content-Type': 'text/plain'});
		res.status(500).send(
			"Your IP could not be determined. This server is misconfigured."
		);
		return;
	}
	req.ident = caps.lookup_ident(ip);
	if (req.ident.ban)
		return res.sendStatus(500);
	next();
});

const vanillaHeaders = {
	'Content-Type': 'text/html; charset=UTF-8',
	'X-Frame-Options': 'sameorigin'
};
const noCache = {
	'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
	'Cache-Control': 'no-cache, no-store'
};
const noCacheHeaders = _.extend(vanillaHeaders, noCache);
const JSONHeaders = _.extend(noCache, {
	'Access-Control-Allow-Origin': '*',
	'Content-Type': 'application/json; charset=UTF-8'
});

app.get('/', function(req, res) {
	res.redirect(`/${config.DEFAULT_BOARD}/`)
});

// Redirect /board to /board/
app.get(/^\/(\w+)$/, function(req, res) {
	res.redirect(`/${req.params[0]}/`);
});

// /board/ pages
app.get(/^\/(\w+)\/$/, function(req, res) {
	res.set(noCacheHeaders);
	const board = req.params[0];
	if (!caps.can_access_board(req.ident, board))
		return res.sendStatus(404);

	let yaku = new db.Yakusoku(board, req.ident);
	yaku.get_tag(-1);
	new Render(yaku, req, res, {
		fullLinks: true,
		board,
		isThread: false
	});
	yaku.once('begin', function (thread_count) {
		yaku.emit('top', page_nav(thread_count, -1, board === 'archive'));
	});
	yaku.once('end', function() {
		yaku.emit('bottom');
		res.end();
		yaku.disconnect();
	});
	yaku.once('error', function(err) {
		winston.error('index:' + err);
		res.end();
		yaku.disconnect();
	});
});

// /board/page* pages
app.get(/^\/(\w+)\/page(\d+)$/,
	function(req, res, next) {
		res.set(noCacheHeaders);
		const board = req.params[0];
		if (!caps.can_access_board(req.ident, board))
			return res.sendStatus(404);

		const page = parseInt(req.params[1], 10);
		let yaku = new db.Yakusoku(board, req.ident);
		yaku.get_tag(page);
		yaku.once('nomatch', function() {
			res.status(302).redirect('.');
			yaku.disconnect();
		});
		yaku.once('begin', function(threadCount) {
			res.opts = {
				board,
				page,
				yaku,
				threadCount
			};
			next();
		});
	}, 
	function(req, res, next) {
		let opts = res.opts;
		const board = opts.board,
			page = opts.page;
		let yaku = opts.yaku;
		
		new Render(yaku, req, res, {
			fullLinks: true,
			board,
			isThread: false
		});
		yaku.emit('top', 
			page_nav(opts.threadCount, page, board === 'archive')
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

function finish(req, res) {
	res.opts.yaku.disconnect();
	res.end();
}

// Thread pages
app.get(/^\/(\w+)\/(\d+)/,
	function(req, res, next) {
		const board = req.params[0];
		if (!caps.can_access_board(req.ident, board))
			return res.sendStatus(404);
		const num = parseInt(req.params[1], 10);
		if (!num)
			return res.sendStatus(404);

		let op;
		if (board === 'graveyard')
			op = num;
		else {
			op = db.OPs[num];
			if (!op)
				return res.sendStatus(404);
			if (!db.OP_has_tag(board, op)) {
				let tag = db.first_tag_of(op);
				if (tag) {
					if (!caps.can_access_board(req.ident, tag))
						return res.sendStatus(404);
					return redirect_thread(res, num, op, tag);
				}
				else {
					winston.warn(`Orphaned post ${num} with tagless OP ${op}`);
					return res.sendStatus(404);
				}
			}
			if (op != num)
				return redirect_thread(res, num, op);
		}
		if (!caps.can_access_thread(req.ident, op))
			return res.sendStatus(404);

		let yaku = new db.Yakusoku(board, req.ident),
			reader = new db.Reader(yaku),
			opts = {redirect: true};

		const lastN = detect_last_n(req.query);
		if (lastN)
			opts.abbrev = lastN + state.hot.ABBREVIATED_REPLIES;

		if (caps.can_administrate(req.ident) && 'reported' in req.query)
			opts.showDead = true;
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
			if (!config.DEBUG && preThread.hctr) {
				// XXX: Always uses the hash of the default language in the etag
				let etag = `W/${preThread.hctr}-`
					+ RES['indexHash-' + config.DEFAULT_LANG];
				const chunks = req.cookies,
					thumb = chunks.thumb;
				if (thumb && common.thumbStyles.indexOf(thumb) >= 0)
					etag += '-' + thumb;
				const etags = ['spoil', 'agif', 'rtime', 'linkify', 'lang'];
				for (let i = 0, l = etags.length; i < l; i++) {
					const tag = etags[i];
					if (chunks[tag])
						etag += `-${tag}-${chunks[tag]}`;
				}
				if (lastN)
					etag += '-last' + lastN;
				if (preThread.locked)
					etag += '-locked';
				if (req.ident.auth)
					etag += '-auth';

				let info = {
					etag: etag,
					req: req
				};
				hooks.trigger_sync('buildETag', info);

				if (req.headers['if-none-match'] === info.etag) {
					yaku.disconnect();
					return res.sendStatus(304);
				}
				res.set(_.extend(vanillaHeaders ,{
					ETag: info.etag,
					'Cache-Control': 'private, max-age=0, must-revalidate'
				}));
			}
			else
				res.set(noCacheHeaders);

			res.opts = {
				board,
				op,
				subject: preThread.subject,
				yaku,
				reader,
				abbrev: opts.abbrev
			};
			next();
		});
	},
	function(req, res, next) {
		let opts = res.opts,
			reader = opts.reader;
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
		opts.yaku.once('error', on_err);

		function on_err(err) {
			winston.error(`thread ${num}:`, err);
			next();
		}
	},
	finish
);

// Pack page navigation data in an object
function page_nav(thread_count, cur_page, ascending) {
	const page_count = Math.max(
		Math.ceil(thread_count / state.hot.THREADS_PER_PAGE)
	);
	return {
		pages: page_count,
		threads: thread_count,
		cur_page: cur_page,
		ascending: ascending
	};
}

// Redirects '/board/num', when num is reply number, not thread
function redirect_thread(res, num, op, tag) {
	if (!tag)
		res.redirect(301, `./${op}#${num}`);
	else
		res.redirect(301, `../${tag}/${op}#${num}`);
}

function detect_last_n(query) {
	if (query.last) {
		const n = parseInt(query.last);
		if (common.reasonable_last_n(n))
			return n;
	}
	return 0;
}
