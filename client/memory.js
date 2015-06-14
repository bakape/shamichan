/*
 * Self-expiring localStorage key controller
 */

var $ = require('jquery'),
	_ = require('underscore');

function Kioku(key, expiry) {
	this.key = key;
	this.expiry = expiry;
}
module.exports = Kioku;

Kioku.prototype.bake_cookie = function (o) {
	var nums = Object.keys(o);
	nums.sort(function (a, b) {
		return parseInt(a, 10) - parseInt(b, 10);
	});
	return nums.join(',');
};

Kioku.prototype.now = function () {
	return Math.floor(Date.now() / 1000);
};

Kioku.prototype.purge_all = function () {
	localStorage.removeItem(this.key);
	$.cookie(this.key, null);
};

Kioku.prototype.purge_expired = function () {
	if (!this.expiry)
		return;
	const o = this.read_all(),
		now = this.now();
	let expired = [];
	for (let k in o) {
		const time = o[k];
		// TEMP cleanup
		if (time === true) {
			expired.push(k);
			continue;
		}
		if (time && now > time + 60*60*24*this.expiry)
			expired.push(k);
	}
	if (expired.length) {
		for (let i = 0, lim = expired.length; i < lim; i++) {
			delete o[expired[i]];
		}
		this.write_all(o);
	}
};

Kioku.prototype.purge_expired_soon = function () {
	setTimeout(this.purge_expired.bind(this),
		5000 + Math.floor(Math.random() * 5000)
	);
};

Kioku.prototype.read_all = function () {
	const key = localStorage.getItem(this.key);
	if (!key)
		return {};
	const val = JSON.parse(key);
	return _.isObject(val) ? val : {};
};

Kioku.prototype.size = function() {
	return _.size(this.read_all());
};

Kioku.prototype.write = function (k, v) {
	// XXX race, would need lock if highly contended
	var o = this.read_all();
	o[k] = v;
	this.write_all(o);
	// Return number of keys
	return _.size(o);
};

Kioku.prototype.write_all = function (o) {
	if (_.isEmpty(o)) {
		this.purge_all();
		return;
	}
	localStorage.setItem(this.key, JSON.stringify(o));
	var baked = this.bake_cookie(o);
	if (baked)
		$.cookie(this.key, baked, {expires: this.expiry});
};
