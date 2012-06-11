var config = require('./config');
var common = require('./common');

var mnemonicStarts = ',k,s,t,d,n,h,b,p,m,f,r,g,z,l,ch'.split(',');
var mnemonicEnds = "a,i,u,e,o,ā,ī,ū,ē,ō,ya,yi,yu,ye,yo,'".split(',');

function ip_mnemonic(header, data) {
	var mnemonic = data.ip;
	if (!mnemonic)
		return header;
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
		header.push(' ' + mnemonic);
	return header;
}

if (typeof AUTH != 'undefined') {
	/* client */
	oneeSama.hook('headerName', ip_mnemonic);
}
else {
	exports.ip_mnemonic = ip_mnemonic;
}
