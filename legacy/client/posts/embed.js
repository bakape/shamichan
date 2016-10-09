// PASTEBIN
const pastebin_re = exports.pastebin_re = /(?:>>>*?)?(?:https?:\/\/)?(?:www\.|m.)?pastebin\.com\/(.*)/;
//Pastebin's API seems built for MAKING pastebins but not sharing them

$(document).on('click', '.pastebin', function(event){
	if (event.which > 1 || event.ctrlKey || event.altKey || event.shiftKey || event.metaKey)
		return;
	var $target = $(event.target);

	var $obj = $target.find('iframe');
	if ($obj.length) {
		$obj.siblings('br').andSelf().remove();
		$target.css({
			width: 'auto',
			height: 'auto'
		});
		return false;
	}

	var m = $target.attr('href').match(pastebin_re);
	if (!m)
		return;
	var $window = $(window),
		width = Math.round($window.innerWidth() * 0.65),
		height = Math.round($window.innerHeight() * 0.65);
	$target
		.css({
			width: width,
			height: height
		})
		.append('<br>', $('<iframe></iframe>', {
			type: 'text/html',
			src: 'https://pastebin.com/embed_iframe.php?i='+ m[1],
			frameborder: '0',
			width: width,
			height: height
		}))
	return false;
});
