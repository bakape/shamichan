(function () {
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
		pick_random($('hr.sectionHr, aside, h1, fieldset'), 0.2).remove();
		setTimeout(shut_down_everything, 500);
	}

	function shut_down_something() {
		var $posts = $('article');
		if (!$posts.length)
			return setTimeout(shut_down_everything, 500);
		var $posts = pick_random($posts, 0.1);
		$posts.each(function () {
			var $post = $(this);
			var $section = $post.closest('section');
			try {
				var thread = Threads.get(extract_num($section));
				var replies = thread.get('replies');
				var num = extract_num($post);
				clear_post_links(replies.get(num), replies);
			}
			catch (e) {}
		});
		$posts.remove();
		if (Math.random() < 0.2)
			pick_random($('figure, blockquote, b'), 0.002).remove();
		setTimeout(shut_down_something, 500);
	}

	var tearingDown = false;
	dispatcher[DEF.TEARDOWN] = function () {
		if (tearingDown)
			return;
		tearingDown = true;
		window.onbeforeunload = null;
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
