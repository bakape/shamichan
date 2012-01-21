(function () {
	function audio(vid, options) {
		var $body = $('body');
		if ($body.data('vid') == vid)
			return;
		$body.data({vid: vid}).find('.audio').remove();
		if (!options)
			options = {autoplay: '1', loop: '1'};
		make_video(vid, options).css({'margin-left': '-9001px', 'position': 'absolute'}).addClass('audio').prependTo($body);
	}
	audio('BnC-cpUCdns');
})();
