/*
 Dice rolls, coin flips, 8balls, syncwatch, banners, JS injections, missle
 launchers - amusement.
 */

const common = require('../common/index'),
	config = require('../config'),
	db = require('../db'),
	fs = require('fs'),
	hooks = require('../util/hooks'),
	hot  = require('./state').hot,
	push = require('./okyaku').push;

let radio;
if (config.RADIO)
	radio = require('./radio');

const rollLimit = 5;
let r = global.redis;

// Load counter from redis on server boot
let pyu_counter = 0;
r.get('pCounter', function(err, res){
	if (err)
		throw err;
	if (res)
		pyu_counter = parseInt(res, 10);
});

exports.getPyu = () => pyu_counter

function roll_dice(frag, post) {
	if (!frag.length)
		return;
	const ms = frag.split(common.dice_re);
	let dice = [];
	for (let i = 1; i < ms.length && dice.length < rollLimit; i += 2) {
		let info = common.parse_dice(ms[i]);
		if (!info)
			continue;
		let rolls = [];
		switch (info.type) {
			case 'pyu':
				if (info.increment) {
					pyu_counter++;
					r.incr('pCounter');
				}
				rolls.push(pyu_counter);
				break;
			case 'radioQueue':
				if (radio)
					rolls.push(radio.queue);
				break;
			case 'syncwatch':
				rolls.push(info);
				break;
			default:
				// At the momement of writing V8 does not support ES6
				// blockscoped declarations in switch statements, thus `var`
				var f = info.faces;
				rolls.push(f);
				rolls.push(info.bias || 0);
				for (let j = 0; j < info.n; j++)
					rolls.push(Math.floor(Math.random() * f) + 1);
		}
		dice.push(rolls);
	}

	if (dice.length)
		post.dice = dice;
}
exports.roll_dice = roll_dice;

function parseDice(post) {
	let dice = post.dice;
	if (!dice)
		return;
	try {
		for (let i = 0, l = dice.length; i < l; i++) {
			dice[i] = JSON.parse(dice[i]);
		}
	}
	catch (e) {
		delete post.dice;
	}
}
exports.parseDice = parseDice;

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
	if (!hot.inject_js)
		return cb();
	fs.readFile(hot.inject_js, {encoding: 'utf8'}, (err, js) => {
		if (err) {
			winston.error('Failed ro read JS injection:', err);
			return cb();
		}
		cb(js);
	});
}

// Push injection to all clients on hot reload
function pushJS() {
	readJS(js => js && push([0, common.EXECUTE_JS, js]));
}
exports.pushJS = pushJS;
