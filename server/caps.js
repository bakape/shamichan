var common = require('../common'),
    config = require('../config'),
    db = require('../db');

exports.can_access = function (ident, board) {
	if (is_admin_ident(ident))
		return true; // including graveyard
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

var mnemonicStarts = ',k,s,t,d,n,h,b,p,m,f,r,g,z,l,ch'.split(',');
var mnemonicEnds = "a,i,u,e,o,ā,ī,ū,ē,ō,ya,yi,yu,ye,yo,'".split(',');

function ip_mnemonic(header, data) {
	var mnemonic = data.ip;
	var nums = mnemonic.split('.');
	if (config.IP_MNEMONIC && nums.length == 4) {
		mnemonic = '';
		for (var i = 0; i < 4; i++) {
			var n = parseInt(nums[i], 10);
			var s = mnemonicStarts[Math.floor(n / 16)] +
					mnemonicEnds[n % 16];
			mnemonic += s;
		}
		header.push(common.safe(' <span title="'+escape(data.ip)+'">'),
				mnemonic, common.safe('</span>'));
	}
	else
		header.push(' ' + data.ip);
	return header;
}

function denote_priv(header, data) {
	if (data.priv)
		header.push(' (priv)');
	return header;
}

exports.augment_oneesama = function (oneeSama, ident) {
	if (is_mod_ident(ident))
		oneeSama.hook('header', ip_mnemonic);
	if (is_admin_ident(ident))
		oneeSama.hook('header', denote_priv);
};
