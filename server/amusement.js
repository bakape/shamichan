/*
 Dice rolls, coin flips, 8balls, syncwatch, banners, JS injections, missle
 launchers - amusement.
 */

let common = require('../common/index'),
	config = require('../config'),
	db = require('../db'),
	hooks = require('../util/hooks');

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
	let client = info.client;
	client.db.get_banner(function (err, msg) {
		if (err)
			return cb(err);
		if (msg)
			client.send([0, common.UPDATE_BANNER, msg]);
		cb(null);
	});
});
