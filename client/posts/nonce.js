/*
Cryptographic nonces for websocket transactions
 */

let main = require('../main'),
	{common, state} = main;

let nonceCache = {};

function get() {
	var nonces;
	if (window.localStorage) {
		try {
			nonces = JSON.parse(localStorage.postNonces);
		}
		catch (e) {}
	}
	else
		nonces = nonceCache;
	return nonces || {};
}
main.reply('nonce:get', get);

function save_nonces(nonces) {
	if (window.localStorage)
		localStorage.postNonces = JSON.stringify(nonces);
	else
		nonceCache = nonces;
}

function today_id() {
	return Math.floor(Date.now() / (1000*60*60*24));
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
main.reply('nonce:create', create);

// Expire old nonces
setTimeout(function() {
	if (!window.localStorage)
		return;
	// we need a lock on postNonces really
	let nonces = get();

	// people messing with their system clock will mess with expiry, doh
	let changed;
	const yesterday = today_id() - 1;
	for (let nonce in nonces) {
		if (nonces[nonce].day >= yesterday)
			continue;
		delete nonces[nonce];
		changed = true;
	}

	if (changed)
		save_nonces(nonces);
}, Math.floor(Math.random()*5000));

function destroy(nonce) {
	// delete only after a delay so all tabs notice that it's ours
	setTimeout(function() {
		let nonces = get();
		if (!nonces[nonce])
			return;
		delete nonces[nonce];
		save_nonces(nonces);
	}, 10000);
}
main.reply('nonce:destroy', destroy);
