/*
 Dice rolls, coin flips, 8balls, syncwatch, banners, JS injections, missle
 launchers - amusement.
 */

const common = require('../common/index'),
	config = require('../config'),
	db = require('../db'),
	fs = require('fs'),
	hooks = require('../util/hooks'),
	state  = require('./state'),
	push = require('./websockets').push,
	radio = config.RADIO && require('./radio')



// Insert #hash commands as tuples into the text body array
function roll_dice(frag, parsed) {
	if (!frag.length)
		return false
	let info
	const types = common.tupleTypes
	switch (frag) {
		case '#flip':
			info = [types.flip, Math.random() > 0.5]
			break
		case '#8ball':
			info = [types.dice, roll(state.hot.EIGHT_BALL.length)]
			break
		case '#q':
			info = radio && [types.radioQueue, radio.queue]
			break
		default:
			info = parseRegularDice(frag) || parseSyncwatch(frag)
	}
	return info && parsed.push(info)
}
exports.roll_dice = roll_dice

function roll(faces) {
	return Math.floor(Math.random() * faces)
}

function parseRegularDice(frag) {
	const m = frag.match(/^#(\d*)d(\d+)([+-]\d+)?$/i)
	if (!m)
		return false
	const n = parseInt(m[1], 10) || 1,
		faces = parseInt(m[2], 10),
		bias = parseInt(m[3] || 10) || 0
	if (n < 1 || n > 10 || faces < 2 || faces > 100)
		return false
	const die = [common.tupleTypes.dice, n, faces, bias]
	for (let i = 0; i < n; i++) {
		info.push(roll(faces) + 1)
	}
	return die
}

function parseSyncwatch(frag) {
	// First capture group may or may not be present
	const sw = frag.match(/^#sw(\d+:)?(\d+):(\d+)([+-]\d+)?$/i)
	if (!sw)
		return false
	const hour = parseInt(sw[1], 10) || 0,
		min = parseInt(sw[2], 10),
		sec = parseInt(sw[3], 10)
	let start = common.serverTime()

	// Offset the start. If the start is in the future, a countdown will be
	// displayed.
	if (sw[4]) {
		const symbol = sw[4].slice(0, 1),
			offset = sw[4].slice(1) * 1000
		start = symbol == '+' ? start + offset : start - offset
	}
	const end = ((hour * 60 + min) * 60 + sec) * 1000 + start

	return [common.tupleTypes.syncwatch, sec, min, hour, start, end]
}

// Information banner
hooks.hook('clientSynced', function (info, cb) {
	const {client} = info;
	client.db.get_banner(function (err, msg) {
		if (err)
			return cb(err);
		if (msg)
			client.send([0, common.UPDATE_BANNER, msg]);
		cb();
	});
});

// Inject JS on client synchronisation
hooks.hook('clientSynced', (info, cb) => {
	readJS(js => {
		if (!js)
			return cb();
		info.client.send([0, common.EXECUTE_JS, js]);
		cb();
	});
});

function readJS(cb) {
	const {inject_js} = state.hot
	if (!inject_js)
		return cb()
	fs.readFile(inject_js, {encoding: 'utf8'}, (err, js) => {
		if (err) {
			winston.error('Failed ro read JS injection:', err)
			return cb()
		}
		cb(js)
	});
}

// Push injection to all clients on hot reload
function pushJS() {
	readJS(js => js && push([0, common.EXECUTE_JS, js]));
}
exports.pushJS = pushJS;

// Regex replacement filter
function hot_filter(frag) {
	let filter = state.hot.FILTER
	if (!filter)
		return frag
	for (let f of filter) {
		const m = frag.match(f.p)
		if (m) {
			// Case sensitivity
			if (m[0].length > 2) {
				if (/[A-Z]/.test(m[0].charAt(1)))
					f.r = f.r.toUpperCase()
				else if (/[A-Z]/.test(m[0].charAt(0)))
					f.r = f.r.charAt(0).toUpperCase() + f.r.slice(1)
			}
			return frag.replace(f.p, f.r)
		}
	}
	return frag
}
exports.hot_filter = hot_filter
