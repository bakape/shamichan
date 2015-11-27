/*
 Serve the HTML part of pages
 */

const _ = require('underscore'),
	cache = require('../../db/cache'),
	caps = require('../caps'),
	common = require('../../common'),
	config = require('../../config'),
	etc = require('../../util/etc'),
	express = require('express'),
	path = require('path'),
	r = require('rethinkdb'),
	{rcon} = global,
	Reader = require('../../db/reader'),
	render = require('../render'),
	state = require('../state'),
	util = require('./util'),
	uaParser = require('ua-parser-js'),
	winston = require('winston')

export default const router = express.Router()
const RES = state.resources
const vanillaHeaders = {
	'Content-Type': 'text/html; charset=UTF-8',
	'X-Frame-Options': 'sameorigin',
	'Cache-Control': 'no-cache, no-store, max-age=0, must-revalidate, private',
	'Pragma': 'no-cache',
	'Expires': 'Fri, 01 Jan 1990 00:00:00 GMT'
}

// Redirect to frontpage, if set, or the default board
router.get('/', (req, res) => {
	if (state.hot.frontpage)
		return res.sendFile(path.resolve(state.hot.frontpage))
	res.redirect(301, `/${config.DEFAULT_BOARD}/`)
})

// Redirect `/board` to `/board/` The client parses the URL to determine
// what page it is on. So we need the trailing slash for easier board
// determination and consistency.
router.get(/^\/(\w+)$/, (req, res) => res.redirect(`/${req.params[0]}/`))

// Respond to board page requests
router.get(/^\/(\w+)\//$/, (req, res) => {
	const [board] = req.board = req.params
	if (!caps.canAccessBoard(req.ident, board))
		return util.send404(res)
	renderBoard(req, resp).catch(err => {
		winston.error("Board rendering error:", err)
		res.status(500).send(err)
	})
})

/**
 * Render board HTML
 * @param {http.ClientRequest} req
 * @param {http.ServerResponse} res
 * @param {string} board
 */
async function renderBoard(req, res) {
	const {board} = req
	const counter = await r.table('main').get('boardCtrs')
		(board).default(0).run(rcon)
	if (!validateEtag(req, res, counter))
		return
	const json = await new Reader(board, req.ident)
		.getBoard(util.resolveSortingOrder(req.query))
	res.send(render(req, json))
}

// Thread pages
router.get(/^\/(\w+)\/(\d+)$/,
	util.boardAccess,
	function(req, res, next) {
		const {board, ident} = req,
			num = parseInt(req.params[1], 10);
		if (!db.validateOP(num, board))
			return redirectNum(req, res, num) || util.send404(res);
		if (!caps.can_access_thread(ident, num))
			return util.send404(res);

		const yaku = new db.Yakusoku(board, ident),
			reader = new db.Reader(ident),
			opts = {};

		const lastN = detect_last_n(req.query);
		if (lastN)
			opts.abbrev = lastN + state.hot.ABBREVIATED_REPLIES;

		reader.get_thread(num, opts);
		reader.once('nomatch', function() {
			util.send404(res);
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

/**
 * Build an etag and check if it mathces the one provided by the client. If yes,
 * send 304 and return false, otherwise set headers and return true.
 * @param {http.ClientRequest} req
 * @param {http.ServerResponse} res
 * @param {int} ctr - Progress counter of board/thread
 * @returns {boolean}
 */
function validateEtag(req, res, ctr) {
	const etag = parseCookies(req, ctr) + parseUserAgent(req)
	if (config.DEBUG) {
		res.set(util.noCacheHeaders)
		return true
	}

	// Etags match. No need to rerender.
	if (req.headers['If-None-Match'] === etag) {
		res.sendStatus(304)
		return false
	}

	const headers = _.clone(vanillaHeaders)
	headers.ETag = etag
	res.set(headers)
	return true
}

/**
 * Build the main part of the etag
 * @param {http.ClientRequest} req
 * @param {int}	ctr
 * @returns {string}
 */
function buildEtag(req, ctr) {
	const {cookies} = req
	const lang = req.lang = etc.resolveConfig(config.LANGS, cookies.lang,
		config.DEFAULT_LANG)
	return `W/${ctr}-${RES['indexHash-' + lang]}-${lang}`
}

/**
 * Parse user agent and detct mobile devices and clients on retarded browsers
 * @param {http.ClientRequest}
 * @returns {sting} - etag appendage
 */
function parseUserAgent(req) {
	const parsed = uaParser(req.header('user-agent'))
	let etag = ''
	if (parsed.device.type) {
		req.isMobile = true
		etag += '-mobile'
	}
	if (['Chrome', 'Firefox', 'Opera', 'Chromium']
		.indexOf(parsed.browser.name) < 0
	) {
		req.isRetarded = true
		etag += '-isRetarded'
	}
	return etag
}

function detect_last_n(query) {
	if (query.last) {
		const n = parseInt(query.last, 10);
		if (common.reasonable_last_n(n))
			return n;
	}
	return 0;
}
