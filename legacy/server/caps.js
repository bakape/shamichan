/*
Manages client read/write permissions
 */

const common = require('../common/index'),
    config = require('../config'),
    db = require('../db'),
	state = require('./state')

/**
 * Confirm client has rights to access board and board is exists
 * @param {Object} ident
 * @param {string} board
 * @returns {boolean}
 */
export function canAccessBoard(ident, board) {
	if (board == config.STAFF_BOARD && !common.checkAuth('janitor', ident))
		return false
    return !ident.ban && config.BOARDS.indexOf(board) >= 0
}

/**
 * Construct the ident object of the client
 * @param {string} ip
 * @returns {Object}
 */
export function lookUpIdent (ip) {
	const ident = {ip}
	if (ip in state.dbCache.bans)
		ident.ban = true
	return ident
}
