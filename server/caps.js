var db = require('../db');

exports.can_access = function (ident, board) {
	if (exports.is_admin_ident(ident))
		return true; // including graveyard
	return db.is_board(board);
};

exports.is_mod_ident = function (ident) {
	return ident && (ident.auth === 'Admin' || ident.auth === 'Moderator');
};

exports.is_admin_ident = function (ident) {
	return ident && ident.auth === 'Admin';
};
