(function () {
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
		imouto.dice = true;
		imouto.queueRoll = queue_roll;
		imouto.allRolls = {sent: 0, seen: 0};
	});

	oneeSama.hook('insertOwnPost', function (extra) {
		if (!postForm || !postForm.imouto || !extra || !extra.dice)
			return;
		var rolls = postForm.imouto.allRolls;
		for (var i = 0; i < extra.dice.length; i++) {
			var n = rolls.seen++;
			var info = rolls[n];
			if (!info)
				info = rolls[n] = {};
			info.dice = extra.dice[i];
			if (info.$tag){
				var r= readable_dice(info.bit, info.dice);
				info.$tag.html(r.safe ? r.safe : r);
			}
		}
	});

	dispatcher[DEF.EXECUTE_JS] = function (msg, op) {
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
