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

function denote_priv(info) {
	if (info.data.priv)
		info.header.push(' (priv)');
}

function dead_media_paths(paths) {
	paths.src = '../dead/src/';
	paths.thumb = '../dead/thumb/';
}

exports.augment_oneesama = function (oneeSama, opts) {
	var ident = opts.ident;
	if (is_mod_ident(ident))
		oneeSama.hook('headerName', authcommon.ip_mnemonic);
	if (is_admin_ident(ident))
		oneeSama.hook('headerName', denote_priv);
	if (is_admin_ident(ident) && opts.board == 'graveyard')
		oneeSama.hook('mediaPaths', dead_media_paths);
};

function parse_ip(ip) {
	var m = ip.match(/^(\d+)\.(\d+)\.(\d+)\.(\d+)(?:\/(\d+))?$/);
	if (!m)
		return false;
	// damn you signed int32s!
	var num = 0;
	for (var i = 4, shift = 1; i > 0; i--) {
		num += parseInt(m[i], 10) * shift;
		shift *= 256;
	}
	var info = {ip: num};
	if (m[5]) {
		var bits = parseInt(m[5], 10);
		if (bits > 0 && bits <= 32)
			info.mask = 0x100000000 - Math.pow(2, 32 - bits);
	}
	return info;
}
exports.parse_ip = parse_ip;
