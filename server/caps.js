var db = require('../db');

function can_access(auth, board) {
	if (auth && auth.auth == 'Admin' && board == 'graveyard')
		return true;
	return db.is_board(board);
}
exports.can_access = can_access;
