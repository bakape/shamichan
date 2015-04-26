/*
 * Dice rolls and fun JS injections
 */

var $ = require('jquery'),
	common = require('../common/index'),
	main = require('./main'),
	state = require('./state');

// Render dice rolls and other hash commands
main.oneeSama.hook('imouto', function (imouto) {
	imouto.dice = true;
	imouto.queueRoll = function(bit) {
		var n = this.allRolls.sent++;
		var info = this.allRolls[n];
		if (!info)
			info = this.allRolls[n] = {};
		info.bit = bit;
		info.$tag = $(this.callback(common.safe('<strong>')));
		this.strong = true;
		this.callback(info.dice ? common.readable_dice(bit, info.dice) : bit);
		this.strong = false;
		this.callback(common.safe('</strong>'));
	};
	imouto.allRolls = {sent: 0, seen: 0};
});

// Handle dice in the postForm
main.oneeSama.hook('insertOwnPost', function (extra) {
	if (!main.postForm || !main.postForm.imouto || !extra || !extra.dice)
		return;
	var rolls = main.postForm.imouto.allRolls;
	extra.dice.forEach(function(dice) {
		var n = rolls.seen++,
			info = rolls[n];
		if (!info)
			info = rolls[n] = {};
		info.dice = dice;
		if (info.$tag){
			const r = common.readable_dice(info.bit, info.dice);
			info.$tag.html(r.safe ? r.safe : r);
		}
	});
});

// Execute server-sent JS in fun threads
main.dispatcher[common.EXECUTE_JS] = function (msg, op) {
	if (state.page.get('thread') != op)
		return;
	try {
		eval(msg[0]);
	}
	catch (e) {
		console.error(e);
	}
};
