/*
Cryptographic nonces for websocket transactions
 */
'use strict';

var common = require('../../common'),
	state = require('../state');

exports.nonces = {};


function get() {
	var nonces;
	if (window.localStorage) {
		try {
			nonces = JSON.parse(localStorage.postNonces);
		}
		catch (e) {}
	}
	else {
		nonces = exports.nonces;
	}
	return nonces || {};
}
exports.get = get;

function save_nonces(nonces) {
	if (window.localStorage)
		localStorage.postNonces = JSON.stringify(nonces);
	else
		exports.nonces = nonces;
}

function today_id() {
	return Math.floor(new Date().getTime() / (1000*60*60*24));
}


function create() {
	const nonces = get(),
		nonce = common.random_id();
	nonces[nonce] = {
		tab: state.page.get('tabID'),
		day: today_id()
	};
	save_nonces(nonces);
	return nonce;
}
exports.create = create;

function expire_nonces() {
	if (!window.localStorage)
		return;
	// we need a lock on postNonces really
	var nonces = get();

	// people messing with their system clock will mess with expiry, doh
	var changed = false;
	const yesterday = today_id() - 1;
	for (var nonce in nonces) {
		if (nonces[nonce].day >= yesterday)
			continue;
		delete nonces[nonce];
		changed = true;
	}

	if (changed)
		save_nonces(nonces);
}
setTimeout(expire_nonces, Math.floor(Math.random()*5000));


function destroy(nonce) {
	var nonces = get();
	if (!nonces[nonce])
		return;
	delete nonces[nonce];
	save_nonces(nonces);
}
exports.destroy = destroy;
