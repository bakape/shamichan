var common = require('./common');

exports.roll_dice = function (frag, post, extra) {
	var ms = frag.split(common.dice_re);
	var dice = [];
	for (var i = 1; i < ms.length; i += 2) {
		var info = common.parse_dice(ms[i]);
		if (!info)
			continue;
		var f = info.faces;
		var rolls = [f];
		for (var j = 0; j < info.n; j++)
			rolls.push(Math.floor(Math.random() * f) + 1);
		if (info.bias)
			rolls.push({bias: info.bias})
		dice.push(rolls);
	}
	if (dice.length) {
		extra.new_dice = dice;
		dice = post.dice ? post.dice.concat(dice) : dice;
		post.dice = dice;
		// Would prefer an appending scheme for adding new rolls but
		// there's no hash value append redis command...
		// I don't want to spill into a separate redis list.
		// Overwriting the whole log every time is quadratic though.
		// TODO: Limit the total number of dice rolls per post
	}
};

exports.inline_dice = function (post, dice) {
	if (dice && dice.length) {
		dice = JSON.stringify(dice);
		post.dice = dice.substring(1, dice.length - 1);
	}
};

exports.extract_dice = function (post) {
	if (!post.dice)
		return;
	try {
		post.dice = JSON.parse('[' + post.dice + ']');
	}
	catch (e) {
		delete post.dice;
	}
};
