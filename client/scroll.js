var lockTarget, lockKeyHeight;
var $lockTarget, $lockIndicator;

var nestLevel = 0;

function with_dom(func) {
try {
	nestLevel++;

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
		if (height > lockHeight - 10)
			window.scrollBy(0, height - lockHeight + 10);
	}
	else if (locked && lockTarget == locked) {
		var newY = $post[0].getBoundingClientRect().top;
		window.scrollBy(0, newY - lockHeight);
	}

	return ret;

} finally {
	nestLevel--;
	if (!nestLevel)
		Backbone.trigger('flushDomUpdates');
}
}

function set_lock_target(num) {
	if (!num && at_bottom())
		num = PAGE_BOTTOM;
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

	var $ind = $lockIndicator;
	if ($ind) {
		$ind.css({visibility: lockTarget ? 'visible' : 'hidden'});
		if (bottom)
			$ind.text('Locked to bottom');
		else if (num) {
			$ind.empty().append($('<a/>', {
				text: '>>' + num,
				href: '#' + num,
			}));
		}
	}
}

oneeSama.hook('menuOptions', function (info) {
	var opts = info.options;
	if (lockTarget && lockTarget == info.num)
		opts.splice(opts.indexOf('Focus'), 1, 'Unfocus');
});

var at_bottom = function() {
	return window.scrollY + window.innerHeight >= $DOC.height();
}
if (window.scrollMaxY !== undefined)
	at_bottom = function () {
		return window.scrollMaxY <= window.scrollY;
	};

(function () {
	menuHandlers.Focus = function ($post) {
		set_lock_target(extract_num($post));
	};
	menuHandlers.Unfocus = function ($post) {
		set_lock_target(null);
	};

	function scroll_shita() {
		if (!lockTarget || (lockTarget == PAGE_BOTTOM))
			set_lock_target(null);
	}

	if (THREAD) {
		$lockIndicator = $('<span id=lock>Locked to bottom</span>', {
			css: {visibility: 'hidden'},
		}).appendTo('body');
		$DOC.scroll(scroll_shita);
		scroll_shita();
	}
})();
