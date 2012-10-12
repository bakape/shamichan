var config = require('./config');
var common = require('./common');

var delayNames = ['now', 'soon', 'later'];
var delayDurations = {now: 0, soon: 60, later: 20*60};
exports.delayDurations = delayDurations;

var mnemonicStarts = ',k,s,t,d,n,h,b,p,m,f,r,g,z,l,ch'.split(',');
var mnemonicEnds = "a,i,u,e,o,ā,ī,ū,ē,ō,ya,yi,yu,ye,yo,'".split(',');

function ip_mnemonic(info) {
	var header = info.header, ip = info.data.ip;
	if (!ip)
		return;
	var nums = ip.split('.');
	if (config.IP_MNEMONIC && nums.length == 4) {
		var mnemonic = '';
		for (var i = 0; i < 4; i++) {
			var n = parseInt(nums[i], 10);
			var s = mnemonicStarts[Math.floor(n / 16)] +
					mnemonicEnds[n % 16];
			mnemonic += s;
		}
		header.push(common.safe(' <span title="'+escape(ip)+'">'),
				mnemonic, common.safe('</span>'));
	}
	else
		header.push(' ' + ip);
}

if (typeof IDENT != 'undefined') {
	/* client */
	oneeSama.hook('headerName', ip_mnemonic);
}
else {
	exports.ip_mnemonic = ip_mnemonic;
}
