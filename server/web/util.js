/*
 Various utility functions
 */

const caps = require('../caps'),
	etc = require('../../util/etc'),
	path = require('path'),
	{resources} = require('../state')


export function parse_forwarded_for(req, ip) {
	const forwards = req.headers['x-forwarded-for']
	if (!forwarded)
		return ip
	const ips = forwards.split(',')
	if (!ips.length)
		return ip
	const last = ips[ips.length - 1].trim()

	// check that it looks like some kind of IPv4/v6 address
	if (!/^[\da-fA-F.:]{3,45}$/.test(last))
		return ip
	return last
}

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

/**
 * Validate client thread sorting order or return the default - by last bump
 * time
 * @param {Object} query - Object of request query parameters
 * @returns {string} - Sorting ordr to use
 */
export function resolveSortingOrder(query) {
    return etc.resolveConfig(['time', 'bumptime', 'replyCount'],
		query.orderBy, 'bumpTime')
}
