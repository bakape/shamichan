(function () {

var $banner;

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

})();
