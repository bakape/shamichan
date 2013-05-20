(function () {

var preview, previewNum;

$DOC.mousemove(mouse_ugoku);

function mouse_ugoku(event) {
	if (!nashi.hover && /^A$/i.test(event.target.tagName)) {
		var m = $(event.target).text().match(/^>>(\d+)/);
		if (m && preview_miru(event, parseInt(m[1], 10)))
			return;
	}
	if (preview) {
		preview.remove();
		preview = previewNum = null;
	}
}

function preview_miru(event, num) {
	/* If there was an old preview of a different thread, remove it */
	if (num != previewNum) {
		var post = $('#' + num);
		if (!post.length)
			return false;
		if (preview)
			preview.remove();
		var bits = post.children();

		/* stupid hack, should be using views */
		if (bits[0] && $(bits[0]).is('.select-handle'))
			bits = bits.slice(1);

		if (post.is('section'))
			bits = bits.slice(0, 3);
		preview = $('<div class="preview"/>').append(bits.clone());
	}

	var overflow = position_preview(event, preview);

	/* Add it to the page if it's new */
	if (num != previewNum) {
		if (overflow > 0) {
			scale_down_to_fit(preview.find('img'), overflow);
			position_preview(event, preview);
		}
		$(document.body).append(preview);
		previewNum = num;
	}
	return true;
}

function position_preview(event, $el) {
	var width = $el.width();
	var height = $el.height();
	if (height < 5) {
		$el.hide();
		$(document.body).append($el);
		width = $el.width();
		height = $el.height();
		$el.detach().show();
	}
	var x = event.pageX + 20;
	var y = event.pageY - height - 20;
	var $w = $(window);
	var overflow = x + width - $w.innerWidth();
	if (overflow > 0) {
		x = Math.max(0, event.pageX - width - 20);
		overflow = x + width - $w.innerWidth();
	}
	var scrollTop = $w.scrollTop();
	if (y < scrollTop) {
		var newY = event.pageY + 20;
		if (newY + height <= scrollTop + $w.height())
			y = newY;
	}
	$el.css({left: x, top: y});
	return overflow;
}

function scale_down_to_fit($img, amount) {
	var w = $img.width(), h = $img.height();
	if (w - amount > 50) {
		var aspect = h / w;
		w -= amount;
		h = aspect * w;
		$img.width(w).height(h);
	}
}

/* We'll get annoying preview pop-ups on touch screens, so disable it.
   Touch detection is unreliable, so wait for an actual touch event */
document.addEventListener('touchstart', touch_screen_event, false);
function touch_screen_event() {
	nashi.hover = true;
	nashi.shortcuts = true;
	if (preview)
		preview.remove();
	$DOC.unbind('mousemove', mouse_ugoku);
	document.removeEventListener('touchstart', touch_screen_event, false);
}

})();
