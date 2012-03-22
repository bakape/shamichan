var authcommon = require('../authcommon'),
    common = require('../common'),
    config = require('../config'),
    db = require('../db');

exports.can_access = function (ident, board) {
	if (board == 'graveyard' && is_admin_ident(ident))
		return true;
	return db.is_board(board);
};

function is_mod_ident(ident) {
	return ident && (ident.auth === 'Admin' || ident.auth === 'Moderator');
}
exports.is_mod_ident = is_mod_ident;

function is_admin_ident(ident) {
	return ident && ident.auth === 'Admin';
}
exports.is_admin_ident = is_admin_ident;

function denote_priv(header, data) {
	if (data.priv)
		header.push(' (priv)');
	return header;
}

exports.augment_oneesama = function (oneeSama, ident) {
	if (is_mod_ident(ident))
		oneeSama.hook('header', authcommon.ip_mnemonic);
	if (is_admin_ident(ident))
		oneeSama.hook('header', denote_priv);
};
