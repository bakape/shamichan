function Kioku(key, expiry) {
	this.key = key;
	this.expiry = expiry;
}

Kioku.prototype.bake_cookie = function (o) {
	var nums = Object.keys(o);
	nums.sort(function (a, b) {
		return parseInt(a, 10) - parseInt(b, 10);
	});
	return nums.join(',');
};

Kioku.prototype.now = function () {
	return Math.floor(new Date().getTime() / 1000);
};

Kioku.prototype.purge_all = function () {
	localStorage.removeItem(this.key);
	$.cookie(this.key, null);
};

Kioku.prototype.purge_expired = function () {
	if (!this.expiry)
		return;
	var o = this.read_all();
	var now = this.now(), expired = [];
	for (var k in o) {
		var time = o[k];
		// TEMP cleanup
		if (time === true) {
			expired.push(k);
			continue;
		}
		if (time && now > time + 60*60*24*this.expiry)
			expired.push(k);
	}
	if (expired.length) {
		expired.forEach(function (k) {
			delete o[k];
		});
		this.write_all(o);
	}
};

Kioku.prototype.purge_expired_soon = function () {
	var delay = 5000 + Math.floor(Math.random() * 5000);
	setTimeout(this.purge_expired.bind(this), delay);
};

Kioku.prototype.read_all = function () {
	var o;
	try {
		o = JSON.parse(localStorage.getItem(this.key));
	}
	catch (e) {}
	return _.isObject(o) ? o : {};
};

Kioku.prototype.write = function (k, v) {
	// XXX race, would need lock if highly contended
	var o = this.read_all();
	o[k] = v;
	this.write_all(o);
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
