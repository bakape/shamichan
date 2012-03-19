(function () {

var $banner;

dispatcher[UPDATE_BANNER] = function (msg, op) {
	if (THREAD != op)
		return;
	if (!$banner)
		$banner = $('<span id="banner"/>').insertAfter('#lock');
	$banner.text(msg[0]);
};

})();
