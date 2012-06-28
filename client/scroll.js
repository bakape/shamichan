var lockTarget, lockKeyHeight;
var $lockTarget, $lockIndicator;

function with_dom(func) {
	var lockHeight, locked = lockTarget, $post;
	if (locked == PAGE_BOTTOM)
		lockHeight = $DOC.height();
	else if (locked) {
		$post = $('#' + locked);
		var r = $post.length && $post[0].getBoundingClientRect();
		if (r && r.bottom > 0 && r.top < window.innerHeight)
			lockHeight = r.top;
		else
			locked = false;
	}
	var ret = func.call(this);
	if (locked == PAGE_BOTTOM) {
		var height = $DOC.height();
		if (height > lockHeight)
			window.scrollBy(0, height - lockHeight + 1);
	}
	else if (locked && lockTarget == locked) {
		var newY = $post[0].getBoundingClientRect().top;
		window.scrollBy(0, newY - lockHeight);
	}

	return ret;
}

function set_lock_target(num) {
	if (num == lockTarget)
		return;
	lockTarget = num;
	var bottom = lockTarget == PAGE_BOTTOM;
	if ($lockTarget)
		$lockTarget.removeClass('scroll-lock');
	if (num && !bottom)
		$lockTarget = $('#' + num).addClass('scroll-lock');
	else
		$lockTarget = null;
	if ($lockIndicator)
		$lockIndicator.css({visibility: bottom ? 'visible' : 'hidden'});
}

(function () {
	menuHandlers.Focus = function ($post) {
		set_lock_target(extract_num($post));
	};

	var at_bottom = function () {
		return window.scrollY + window.innerHeight >= $DOC.height();
	};
	if (window.scrollMaxY !== undefined)
		at_bottom = function () {
			return window.scrollMaxY <= window.scrollY;
		};

	function scroll_shita() {
		if (!lockTarget || (lockTarget == PAGE_BOTTOM))
			set_lock_target(at_bottom() && PAGE_BOTTOM);
	}

	if (THREAD) {
		$lockIndicator = $('<span id="lock">Locked to bottom</span>'
				).css({visibility: 'hidden'}).appendTo('body');
		$DOC.scroll(scroll_shita);
		scroll_shita();
	}
})();
