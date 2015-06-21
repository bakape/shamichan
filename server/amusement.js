/*
 Dice rolls, coin flips, 8balls, syncwatch, banners, JS injections, missle
 launchers - amusement.
 */
'use strict';

var common = require('../common/index'),
	config = require('../config'),
	db = require('../db'),
	hooks = require('../util/hooks');

var radio;
if (config.RADIO)
	radio = require('./radio');

const rollLimit = 5;
var pyu_counter;
var r = global.redis;

function roll_dice (frag, post, extra) {
	const ms = frag.split(common.dice_re);
	let dice = [];
	for (let i = 1; i < ms.length && dice.length < rollLimit; i += 2) {
		let info = common.parse_dice(ms[i]);
		if (!info)
			continue;
		const f = info.faces;
		let rolls = [];
		// Pyu counter
		if (info.pyu){
			if (info.pyu == 'increment') {
				pyu_counter++;
				r.incr('pCounter');
			}
			rolls.push(pyu_counter);
		}
		// r/a/dio song queue
		else if (info.q && radio)
			rolls.push(radio.queue);
		// Syncwatch
		else if(info.start)
			rolls.push({
				start:info.start,
				end:info.end,
				hour:info.hour,
				min:info.min,
				sec:info.sec
			});
		else {
			rolls.push(f);
			for (let j = 0; j < info.n; j++)
				rolls.push(Math.floor(Math.random() * f) + 1);
		}
		if (info.bias)
			rolls.push({bias: info.bias});
		dice.push(rolls);
	}
	if (dice.length) {
		// Would prefer an appending scheme for adding new rolls but
		// there's no hash value append redis command...
		// I don't want to spill into a separate redis list.
		// Overwriting the whole log every time is quadratic though.
		// Enforcing a roll limit to deter that and for sanity
		let exist = post.dice ? post.dice.length : 0;
		if (dice.length + exist > rollLimit)
			dice = dice.slice(0, Math.max(0, rollLimit - exist));
		if (dice.length) {
			extra.new_dice = dice;
			dice = post.dice ? post.dice.concat(dice) : dice;
			post.dice = dice;
		}
	}
}
exports.roll_dice = roll_dice;

function inline_dice(post, dice) {
	if (dice && dice.length) {
		dice = JSON.stringify(dice);
		post.dice = dice.slice(1, -1);
	}
}

// Load counter from redis on server boot
(function(){
	r.get('pCounter', function(err, res){
		if (err)
			return pyu_counter = false;
		// Initial query
		if (!res)
			return pyu_counter = parseInt(0, 10);
		pyu_counter = parseInt(res, 10);
	});
})();

hooks.hook('attachToPost', function (attached, cb) {
	var new_dice = attached.extra.new_dice;
	if (new_dice) {
		attached.attach.dice = new_dice;
		inline_dice(attached.writeKeys, attached.post.dice);
	}
	cb(null);
});

hooks.hook_sync('inlinePost', function (info) {
	inline_dice(info.dest, info.src.dice);
});

hooks.hook_sync('extractPost', function (post) {
	if (!post.dice)
		return;
	try {
		post.dice = JSON.parse('[' + post.dice + ']');
	}
	catch (e) {
		delete post.dice;
	}
});

// This is looking rather boilerplatey

hooks.hook('clientSynced', function (info, cb) {
	var op = info.op, client = info.client;
	if (op) {
		client.db.get_fun(op, function (err, js) {
			if (err)
				return cb(err);
			if (js)
				client.send([op, common.EXECUTE_JS, js]);
			cb(null);
		});
	}
	else
		cb(null);
});

// Information banner
hooks.hook('clientSynced', function (info, cb) {
	var client = info.client;
	client.db.get_banner(function (err, msg) {
		if (err)
			return cb(err);
		if (msg)
			client.send([0, common.UPDATE_BANNER, msg]);
		cb(null);
	});
});
