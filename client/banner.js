(function () {

var $banner;

dispatcher[UPDATE_BANNER] = function (msg, op) {
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
	if ($banner)
		$banner.text(msg[0]);
};

})();
