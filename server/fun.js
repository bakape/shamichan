(function () {
	var vid = 'BnC-cpUCdns';

	var $target = $('body');
	if ($target.data('vid') == vid)
		return;
	else
		$target.data({vid: vid}).find('object').remove();
	make_video(vid, {autoplay: '1', loop: '1'}).css({'margin-left': '-9001px', 'position': 'absolute'}).appendTo($target);
})();
