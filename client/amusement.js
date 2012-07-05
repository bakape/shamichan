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
		if (info.$tag)
			info.$tag.text(readable_dice(info.bit, info.dice));
	}
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

function game_over() {
	setTimeout(function () {
		location.reload(true);
	}, 2000);
	$DOC.children().remove();
}

function shut_down_everything() {
	var $threads = $('section');
	if (!$threads.length)
		return setTimeout(game_over, 1000);
	pick_random($threads, 0.2).remove();
	pick_random($('hr, aside, h1, fieldset'), 0.2).remove();
	setTimeout(shut_down_everything, 500);
}

function shut_down_something() {
	var $posts = $('article');
	if (!$posts.length)
		return setTimeout(shut_down_everything, 500);
	var $posts = pick_random($posts, 0.1);
	$posts.each(function () {
		var num = extract_num($(this));
		if (CurThread) {
			try {
				clear_post_links(lookup_post(num));
			}
			catch (e) {}
		}
	});
	$posts.remove();
	if (Math.random() < 0.2)
		pick_random($('figure, blockquote, b'), 0.002).remove();
	setTimeout(shut_down_something, 500);
}

var tearingDown = false;
dispatcher[TEARDOWN] = function () {
	if (tearingDown)
		return;
	tearingDown = true;
	shut_down_something();
};

function pick_random($items, proportion) {
	var len = $items.length;
	var origLen = len;
	var toDelete = Math.max(1, Math.min(len, Math.ceil(len * proportion)));
	var $picked = $();
	for (; len > 0 && toDelete > 0; toDelete--) {
		var i = Math.floor(Math.random() * len);
		$picked = $picked.add($items[i]);
		$items.splice(i, 1);
		len = $items.length;
	}
	return $picked;
}

})();
