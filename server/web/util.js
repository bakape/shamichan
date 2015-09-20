/*
 Various utility functions
 */

const caps = require('../caps'),
	path = require('path'),
	{resources} = require('../state');

function parse_forwarded_for(ff) {
	if (!ff)
		return null;
	const ips = ff.split(',');
	if (!ips.length)
		return null;
	const last = ips[ips.length - 1].trim();
	// check that it looks like some kind of IPv4/v6 address
	if (!/^[\da-fA-F.:]{3,45}$/.test(last))
		return null;
	return last;
}
exports.parse_forwarded_for = parse_forwarded_for;

function boardAccess(req, res, next) {
	const board = req.board = req.params[0];
	if (!caps.can_access_board(req.ident, board))
		return send404(res);
	next();
}
exports.boardAccess = boardAccess;

exports.noCacheHeaders = {
	'X-Frame-Options': 'sameorigin',
	'Expires': 'Thu, 01 Jan 1970 00:00:00 GMT',
	'Cache-Control': 'no-cache, no-store'
};

function send404(res) {
	res.status(404).sendFile(path.resolve('www/404.html'));
}
exports.send404 = send404;
