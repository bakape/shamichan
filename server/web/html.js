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
	'Cache-Control': 'max-age=0, must-revalidate',
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
router.get(/^\/(\w+)$/, (req, res) =>
	res.redirect(`/${req.params[0]}/`))

// Respond to board and thread page requests
router.get(/^\/(\w+)\/()/$/, (req, res) => {
	const [board, thread] = req.board = req.params
	if (!caps.canAccessBoard(req.ident, board))
		return util.send404(res)
	const handler = thread
		? renderThread(req, resp, parseInt(thread))
		: renderBoard(req, resp)
	handler.catch(err => {
		winston.error("Rendering error:", err)
		res.status(500).send(err)
	})
})

/**
 * Render board HTML
 * @param {http.ClientRequest} req
 * @param {http.ServerResponse} res
 */
async function renderBoard(req, res) {
	const {board} = req
	const counter = await r.table('main')
		.get('boardCtrs')
		(board)
		.default(0)
		.run(rcon)
	if (!validateEtag(req, res, counter))
		return
	const json = await new Reader(board, req.ident)
		.getBoard(util.resolveSortingOrder(req.query))
	res.send(render(req, json))
}

/**
 * Render thread HTML
 * @param {http.ClientRequest} req
 * @param {http.ServerResponse} res
 * @param {int}	thread
 */
async function renderThread(req, res, thread) {
	const {board} = req
	if (!(await cache.validateOP(thread, board)))
		return util.send404(res)
	const counter = await r.table('threads')
		.get(thread)
		('history')
		.count()

	// Append last N posts to display setting, if valid
	const lastN = detectLastN(req.query)
	if (lastN)
		extra = `-last${lastN}`
	if (!validateEtag(req, res, counter, extra))
		return
	const json = await new Reader(board, req.ident).getThread(thread, lastN)
	res.send(render(req, json))
}


/**
 * Build an etag and check if it mathces the one provided by the client. If yes,
 * send 304 and return false, otherwise set headers and return true.
 * @param {http.ClientRequest} req
 * @param {http.ServerResponse} res
 * @param {int} ctr - Progress counter of board/thread
 * @returns {boolean}
 */
function validateEtag(req, res, ctr, extra) {
	const etag = parseCookies(req, ctr) + parseUserAgent(req)
	if (config.DEBUG) {
		res.set(util.noCacheHeaders)
		return true
	}
	const {auth} = req.ident
	if (auth)
		etag += `-${auth}`
	if (extra)
		etag += extra

	// Etags match. No need to rerender.
	if (req.headers['If-None-Match'] === etag) {
		res.sendStatus(304)
		return false
	}
	const headers = _.clone(vanillaHeaders)
	headers.ETag = etag

	// Don't distribute confidential caches to other clients
	if (auth)
		heders['Cache-Control'] += ', private'
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

/**
 * Validate the client's last N posts to display setting
 * @param {Object} query
 * @returns {int}
 */
function detectLastN(query) {
	if (query.last) {
		const n = parseInt(query.last, 10)
		if (common.reasonable_last_n(n))
			return n
	}
	return 0
}
