(function () {

var $banner;

function queue_roll(bit) {
	var n = this.allRolls.sent++;
	var info = this.allRolls[n];
	if (!info)
		info = this.allRolls[n] = {};
	info.bit = bit;
	info.$tag = $(this.callback(safe('<strong>')));
	this.strong = true;
	this.callback(info.dice ? readable_dice(bit, info.dice) : bit);
	this.strong = false;
	this.callback(safe('</strong>'));
}

oneeSama.hook('imouto', function (imouto) {
	imouto.dice = GAME_BOARDS.indexOf(BOARD) >= 0;
	imouto.queueRoll = queue_roll;
	imouto.allRolls = {sent: 0, seen: 0};
	return imouto;
});

oneeSama.hook('insertOwnPost', function (links, extra) {
	if (!postForm || !postForm.imouto || !extra || !extra.dice)
		return links;
	var rolls = postForm.imouto.allRolls;
	for (var i = 0; i < extra.dice.length; i++) {
		var n = rolls.seen++;
		var info = rolls[n];
		if (!info)
			info = rolls[n] = {};
		info.dice = extra.dice[i];
		if (info.$tag)
			info.$tag.text(readable_dice(info.bit, info.dice));
	}
	return links;
});

dispatcher[UPDATE_BANNER] = function (msg, op) {
	msg = msg[0];
	if (!$banner) {
		var dest;
		if (THREAD == op)
			dest = '#lock';
		else {
			var $s = $('#' + op);
			if ($s.is('section'))
				dest = $s.children('header');
		}
		if (dest)
			$banner = $('<span id="banner"/>').insertAfter(dest);
	}
	if ($banner) {
		if (msg)
			$banner.text(msg);
		else {
			$banner.remove();
			$banner = null;
		}
	}
};

dispatcher[EXECUTE_JS] = function (msg, op) {
	if (THREAD != op)
		return;
	try {
		eval(msg[0]);
	}
	catch (e) {
		/* fgsfds */
	}
};

})();
