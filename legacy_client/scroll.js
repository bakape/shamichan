var lockTarget, lockKeyHeight;
var $lockTarget, $lockIndicator;
var lockedManually;
var dropAndLockTimer;

var nestLevel = 0;

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

	var ret;
	try {
		nestLevel++;
		ret = func.call(this);
	}
	finally {
		if (!--nestLevel)
			Backbone.trigger('flushDomUpdates');
	}

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
}

function set_lock_target(num, manually) {
	lockedManually = manually;

	if (!num && at_bottom())
		num = PAGE_BOTTOM;
	if (num == lockTarget)
		return;
	lockTarget = num;
	var bottom = lockTarget == PAGE_BOTTOM;
	if ($lockTarget)
		$lockTarget.removeClass('scroll-lock');
	if (num && !bottom && manually)
		$lockTarget = $('#' + num).addClass('scroll-lock');
	else
		$lockTarget = null;

	var $ind = $lockIndicator;
	if ($ind) {
		var visible = bottom || manually;
		$ind.css({visibility: visible ? 'visible' : 'hidden'});
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
	if (lockTarget && info.model && lockTarget == info.model.id)
		opts.splice(opts.indexOf('Focus'), 1, 'Unfocus');
});

Backbone.on('hide', function (model) {
	if (model && model.id == lockTarget)
		set_lock_target(null);
});

connSM.on('dropped', function () {
	if (!dropAndLockTimer)
		dropAndLockTimer = setTimeout(drop_and_lock, 10 * 1000);
});

function drop_and_lock() {
	if (connSM.state == 'synced')
		return;
	// On connection drop, focus the last post.
	// This to prevent jumping to thread bottom on reconnect.
	autoUnlock(true);
}

function autoUnlock(unlock){
	if (!CurThread || lockedManually)
		return;
	if (unlock && !options.get('alwaysLock')){
		var last = CurThread.get('replies').last();
		if (last)
			set_lock_target(last.id, false);
	}
	else
		set_lock_target(null);
}

connSM.on('synced', function () {
	// If we dropped earlier, stop focusing now.
	autoUnlock(false);
	if (dropAndLockTimer) {
		clearTimeout(dropAndLockTimer);
		dropAndLockTimer = null;
	}
});

var at_bottom = function() {
	return window.scrollY + window.innerHeight >= $DOC.height() - 5;
};
if (window.scrollMaxY !== undefined)
	at_bottom = function () {
		return window.scrollMaxY <= window.scrollY;
	};

(function () {
	menuHandlers.Focus = function (model) {
		var num = model && model.id;
		set_lock_target(num, true);
	};
	menuHandlers.Unfocus = function () {
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

// Account for banner height, when scrolling to an anchor
function scroll_above_banner(){
	if (/^#\d+$/.test(location.hash))
		$(window).scrollTop($(window).scrollTop()-$('#banner').height());
}

window.onpopstate = scroll_above_banner;
window.onload = scroll_above_banner;
