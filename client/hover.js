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
	var width = preview.width();
	var height = preview.height();
	if (height < 5) {
		preview.hide();
		$(document.body).append(preview);
		width = preview.width();
		height = preview.height();
		preview.detach().show();
	}
	var x = event.pageX + 20;
	var y = event.pageY - height - 20;
	var $w = $(window);
	if (x + width > $w.innerWidth())
		x = Math.max(0, event.pageX - width - 20);
	var scrollTop = $w.scrollTop();
	if (y < scrollTop) {
		var newY = event.pageY + 20;
		if (newY + height <= scrollTop + $w.height())
			y = newY;
	}
	preview.css({left: x, top: y});
	if (num != previewNum) {
		$(document.body).append(preview);
		previewNum = num;
	}
	return true;
}

/* We'll get annoying preview pop-ups on touch screens, so disable it.
   Touch detection is unreliable, so wait for an actual touch event */
document.addEventListener('touchstart', touch_screen_event, false);
function touch_screen_event() {
	nashi.hover = true;
	if (preview)
		preview.remove();
	$DOC.unbind('mousemove', mouse_ugoku);
	document.removeEventListener('touchstart', touch_screen_event, false);
}

})();
