var $DOC = $(document);
var lockedToBottom, lockKeyHeight;

if (window.scrollMaxY !== undefined) {
	function at_bottom() {
		return window.scrollMaxY <= window.scrollY;
	}
}
else {
	function at_bottom() {
		return window.scrollY + window.innerHeight >= $DOC.height();
	}
}

function with_dom(func) {
	var lockHeight, locked = lockedToBottom;
	if (locked)
		lockHeight = $DOC.height();
	var ret = func.call(this);
	if (locked) {
		var height = $DOC.height();
		if (height > lockHeight)
			window.scrollBy(0, height - lockHeight + 1);
	}
	return ret;
}

function scroll_shita() {
	var lock = at_bottom();
	if (lock != lockedToBottom)
		set_scroll_locked(lock);
}

function set_scroll_locked(lock) {
	lockedToBottom = lock;
	$('#lock').css({visibility: lock ? 'visible' : 'hidden'});
}

$(function () {
	if (THREAD) {
		$('<span id="lock">Locked to bottom</span>'
				).css({visibility: 'hidden'}).appendTo('body');
		$DOC.scroll(scroll_shita);
		scroll_shita();
	}
});
