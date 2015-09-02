/*
Manages client read/write permissions
 */

const common = require('../common/index'),
    config = require('../config'),
    db = require('../db'),
	state = require('./state');

function can_access_board(ident, board) {
	if (board == config.STAFF_BOARD && !common.checkAuth('janitor', ident))
		return false;
	if (ident.ban)
		return false;
	return db.is_board(board);
}
exports.can_access_board = can_access_board;

function can_access_thread(ident, op) {
	const board = db.boards[op];
	return board && can_access_board(ident, board);
}
exports.can_access_thread = can_access_thread;

function lookup_ident (ip) {
	const ident = {ip};
	if (ip in state.dbCache.bans)
		ident.ban = true;
	return ident;
}
exports.lookup_ident = lookup_ident;


