// Define vars both on server and client
var _, common, config, DEF,
	isNode = typeof navigator === 'undefined';

if (isNode) {
	_ = require('underscore');
	common = require('../common');
	config = require('../config');
	DEF = exports;
}
else {
	_ = window._;
	common = window;
	config = window.config;
	DEF = window.DEF;
}

DEF.FETCH_ADDRESS = 101;
DEF.SET_ADDRESS_NAME = 102;
DEF.BAN = 103;

var modCache = {}; // TEMP

var suspensionKeys = ['boxes', 'bans', 'slows', 'suspensions', 'timeouts'];

var delayNames = ['now', 'soon', 'later'];
var delayDurations = {now: 0, soon: 60, later: 20*60};

var mnemonicStarts = ',k,s,t,d,n,h,b,p,m,f,r,g,z,l,ch'.split(',');
var mnemonicEnds = "a,i,u,e,o,a,i,u,e,o,ya,yi,yu,ye,yo,'".split(',');

function ip_mnemonic(ip) {
	if (/^[a-fA-F0-9:]{3,45}$/.test(ip))
		return ipv6_mnemonic(ip);
	if (!is_IPv4_ip(ip))
		return null;
	var nums = ip.split('.');
	var mnemonic = '';
	for (var i = 0; i < 4; i++) {
		var n = parseInt(nums[i], 10);
		var s = mnemonicStarts[Math.floor(n / 16)] +
				mnemonicEnds[n % 16];
		mnemonic += s;
	}
	return mnemonic;
}

var ipv6kana = (
	',a,i,u,e,o,ka,ki,' +
	'ku,ke,ko,sa,shi,su,se,so,' +
	'ta,chi,tsu,te,to,na,ni,nu,' +
	'ne,no,ha,hi,fu,he,ho,ma,' +
	'mi,mu,me,mo,ya,yu,yo,ra,' +
	'ri,ru,re,ro,wa,wo,ga,gi,' +
	'gu,ge,go,za,ji,zu,ze,zo,' +
	'da,de,do,ba,bi,bu,be,bo'
).split(',');
if (ipv6kana.length != 64)
	throw new Error('bad ipv6 kana!');

var ipv6alts = {
	sa: 'sha', su: 'shu', so: 'sho',
	ta: 'cha', tsu: 'chu', to: 'cho',
	fu: 'hyu',
	za: 'ja', zu: 'ju', zo: 'jo',
};

function ipv6_mnemonic(ip) {
	var groups = explode_IPv6_ip(ip);
	if (!groups || groups.length != 8)
		return null;

	// takes 8 bits, returns kana
	function p(n) {
		// bits 0-5 are lookup; 6-7 are modifier
		var kana = ipv6kana[n & 0x1f];
		if (!kana)
			return kana;
		var mod = (n >> 6) & 3;
		// posibly modify the kana
		if (mod > 1 && kana[0] == 'b')
			kana = 'p' + kana[1];
		if (mod == 3) {
			var alt = ipv6alts[kana];
			if (alt)
				return alt;
			var v = kana[kana.length - 1];
			if ('knhmrgbp'.indexOf(kana[0]) >= 0 && 'auo'.indexOf(v) >= 0)
				kana = kana[0] + 'y' + kana.slice(1);
		}
		return kana;
	}

	var ks = groups.map(function (hex) {
		var n = parseInt(hex, 16);
		return p(n >> 8) + p(n);
	});

	function cap(s) {
		return s[0].toUpperCase() + s.slice(1);
	}

	// discard first group (RIR etc)
	// also discard some other groups for length/anonymity
	var sur = ks.slice(1, 4).join('');
	var given = ks.slice(6, 8).join('');
	return cap(sur) + ' ' + cap(given);
}

function append_mnemonic(info) {
	var header = info.header, ip = info.data.ip;
	if (!ip)
		return;
	var mnemonic = config.IP_MNEMONIC && ip_mnemonic(ip);
	var key = ip_key(ip);

	// Terrible hack.
	if (mnemonic && modCache.addresses) {
		var addr = modCache.addresses[key];
		if (addr && addr.name && config.IP_TAGGING)
			mnemonic += ' "' + addr.name + '"';
	}

	var s = common.safe;
	var title = mnemonic ? [s(' title="'), ip, s('"')] : '';
	header.push(s(' <a class="mod addr"'), title, s('>'),
			mnemonic || ip, s('</a>'));
}

function denote_hidden(info) {
	if (info.data.hide)
		info.header.push(common.safe(
				' <em class="mod hidden">(hidden)</em>'));
}

function is_IPv4_ip(ip) {
	if (typeof ip != 'string' || !/^\d+\.\d+\.\d+\.\d+$/.exec(ip))
		return false;
	var nums = ip.split('.');
	for (var i = 0; i < 4; i++) {
		var n = parseInt(nums[i], 10);
		if (n > 255)
			return false;
		if (n && nums[i][0] == '0')
			return false;
	}
	return true;
}

var is_valid_ip = function (ip) {
	return typeof ip == 'string' && /^[\da-fA-F.:]{3,45}$/.test(ip);
}

function explode_IPv6_ip(ip) {
	if (typeof ip != 'string')
		return null;

	var groups = ip.split(':');
	var gap = groups.indexOf('');
	if (gap >= 0 || groups.length != 8) {
		// expand ::
		if (gap < 0 || gap != groups.lastIndexOf(''))
			return null;
		var zeroes = [gap, 1];
		for (var i = groups.length; i < 9; i++)
			zeroes.push('0');
		groups.splice.apply(groups, zeroes);
	}

	// check hex components
	for (var i = 0; i < groups.length; i++) {
		var n = parseInt(groups[i], 16);
		if (_.isNaN(n) || n > 0xffff)
			return null;
		groups[i] = n.toString(16);
	}

	return groups;
}

function ip_key(ip) {
	if (!is_IPv4_ip(ip)) {
		// chop off the last half of IPv6 ips
		var bits = explode_IPv6_ip(ip);
		if (bits && bits.length == 8)
			return bits.slice(0, 4).join(':');
	}
	return ip;
}

if (typeof IDENT != 'undefined') {
	/* client */
	window.ip_mnemonic = ip_mnemonic;
	oneeSama.hook('headerName', append_mnemonic);
	oneeSama.hook('headerName', denote_hidden);
}
else if (isNode) {
	exports.ip_mnemonic = ip_mnemonic;
	exports.append_mnemonic = append_mnemonic;
}

if (isNode){
	exports.modCache = modCache;
	exports.suspensionKeys = suspensionKeys;
	exports.delayDurations = delayDurations;
	exports.denote_hidden = denote_hidden;
	exports.is_IPv4_ip = is_IPv4_ip;
	exports.is_valid_ip = is_valid_ip;
	exports.ip_key = ip_key;
}
