var config = require('./config');
var common = require('./common');

var delayNames = ['now', 'soon', 'later'];
var delayDurations = {now: 0, soon: 60, later: 20*60};
exports.delayDurations = delayDurations;

var mnemonicStarts = ',k,s,t,d,n,h,b,p,m,f,r,g,z,l,ch'.split(',');
var mnemonicEnds = "a,i,u,e,o,a,i,u,e,o,ya,yi,yu,ye,yo,'".split(',');

function ip_mnemonic(ip) {
	if (typeof ip != 'string')
		return '<bad IP>';
	var nums = ip.split('.');
	if (nums.length != 4)
		return '<bad IP>';
	var mnemonic = '';
	for (var i = 0; i < 4; i++) {
		var n = parseInt(nums[i], 10);
		var s = mnemonicStarts[Math.floor(n / 16)] +
				mnemonicEnds[n % 16];
		mnemonic += s;
	}
	return mnemonic;
}

function append_mnemonic(info) {
	var header = info.header, ip = info.data.ip;
	if (!ip)
		return;
	if (config.IP_MNEMONIC)
		header.push(common.safe(' <span title="'+escape(ip)+'">'),
				ip_mnemonic(ip), common.safe('</span>'));
	else
		header.push(' ' + ip);
}

if (typeof IDENT != 'undefined') {
	/* client */
	window.ip_mnemonic = ip_mnemonic;
	oneeSama.hook('headerName', append_mnemonic);
}
else {
	exports.append_mnemonic = append_mnemonic;
}
