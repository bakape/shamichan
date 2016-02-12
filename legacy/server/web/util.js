/*
 Various utility functions
 */

const caps = require('../caps'),
	etc = require('../../util/etc'),
	path = require('path'),
	{resources} = require('../state')

export function boardAccess(req, res, next) {
	const board = req.board = req.params[0]
	if (!caps.can_access_board(req.ident, board))
		return send404(res)
	next()
}

export const noCacheHeaders = {
	'X-Frame-Options': 'sameorigin',
	'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
	'Cache-Control': 'no-cache, no-store'
}

/**
 * Send 404 status and the 404 HTML document
 * @param {http.ServerResponse} res
 */
export function send404(res) {
	res.status(404).sendFile(path.resolve('www/404.html'))
}
