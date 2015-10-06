/*
 * Dice rolls and fun JS injections
 */

const main = require('./main'),
	{$, common, state, oneeSama} = main;

// Render dice rolls and other hash commands
oneeSama.hook('imouto', function (imouto) {
	imouto.dice = true;
	imouto.queueRoll = function (bit) {
		const number = this.allRolls.sent++;
		let info = this.allRolls[number];
		if (!info)
			info = this.allRolls[number] = {};
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
oneeSama.hook('insertOwnPost', ({dice}) => {
	let postForm = main.request('postForm');
	if (!postForm || !postForm.imouto || !dice)
		return;
	let rolls = postForm.imouto.allRolls;
	for (let i = 0, lim = dice.length; i < lim; i++) {
		const n = rolls.seen++;
		let info = rolls[n];
		if (!info)
			info = rolls[n] = {};
		info.dice = dice[i];
		if (info.$tag) {
			const r = common.readable_dice(info.bit, info.dice);
			info.$tag.html(r.safe ? r.safe : r);
		}
	}
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
