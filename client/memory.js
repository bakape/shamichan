/*
 * Self-expiring localStorage key controller
 */

import {isObject, isEmpty, size} from 'underscore'
import * as Cookie from 'js-cookie'

export default class Memory {
	constructor(key, expiry, needCookie) {
		this.key = key;
		this.expiry = expiry;
		this.needCookie = needCookie;
		setTimeout(() => this.purgeExpired(), 5000)
	}
	bakeCookie(object) {
		if (!this.needCookie)
			return;
		let nums = Object.keys(object);
		nums.sort(function(a, b) {
			return parseInt(a, 10) - parseInt(b, 10);
		});
		Cookie.set(this.key, nums.join('/'));
	}
	now() {
		return Math.floor(Date.now() / 1000);
	}
	purgeAll() {
		localStorage.removeItem(this.key);
		if (this.needCookie)
			Cookie.remove(this.key);
	}
	readAll() {
		const key = localStorage.getItem(this.key);
		if (!key)
			return {};
		let val;
		try {
			val = JSON.parse(key);
		}
		catch(e) {}
		return isObject(val) ? val : {};
	}
	writeAll(object) {
		if (isEmpty(object))
			return this.purgeAll();
		localStorage.setItem(this.key, JSON.stringify(object));
		this.bakeCookie(object)
	}
	write(key) {
		let object = this.readAll();
		object[key] = this.now();
		this.writeAll(object);
		// Return number of keys
		return size(object);
	}
	size() {
		return size(this.readAll());
	}
	purgeExpired() {
		if (!this.expiry)
			return;
		let object = this.readAll();
		const now = this.now(),
			limit = 86400 * this.expiry;
		let expired = [];
		for (let key in object) {
			const time = object[key];
			if (time && now > time + limit)
				expired.push(key);
		}
		if (!expired.length)
			return;
		for (let i = 0, lim = expired.length; i < lim; i++) {
			delete object[expired[i]];
		}
		this.writeAll(object);
	}
}
