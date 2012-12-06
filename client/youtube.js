var youtube_url_re = /(?:>>>*?)?(?:https?:\/\/)?(?:www\.)?youtube\.com\/watch\/?\?((?:[^\s#&=]+=[^\s#&]*&)*)?v=([\w-]{11})((?:&[^\s#&=]+=[^\s#&]*)*)&?(#t=[\dhms]{1,9})?/;
var youtube_time_re = /^#t=(?:(\d\d?)h)?(?:(\d\d?)m)?(?:(\d\d?)s)?$/;

function make_video(id, params, start) {
	if (!params)
		params = {allowFullScreen: 'true'};
	params.allowScriptAccess = 'always';
	var query = {version: 3, autohide: 1, showinfo: 0, fs: 1,
		modestbranding: 1};
	if (start)
		query.start = start;
	if (params.autoplay)
		query.autoplay = params.autoplay;
	if (params.loop) {
		query.loop = '1';
		query.playlist = id;
	}

	var bits = [];
	for (var k in query)
		bits.push(encodeURIComponent(k) + '=' +
				encodeURIComponent(query[k]));
	var uri = encodeURI('http://www.youtube.com/v/' + id) + '?' +
			bits.join('&');
	var dims = video_dims();
	var $obj = $('<object></object>').attr(dims);
	for (var name in params)
		$obj.append($('<param></param>').attr({name: name,
				value: params[name]}));
	$('<embed></embed>').attr(params).attr(dims).attr({src: uri,
		type: 'application/x-shockwave-flash'}).appendTo($obj);
	return $obj;
}

function video_dims() {
	if (window.screen && screen.width <= 320)
		return {width: 250, height: 150};
	else
		return {width: 560, height: 340};
}

$(document).on('click', '.watch', function (event) {
	if (event.which > 1)
		return;
	var $target = $(event.target);
	var $video = $target.find('object');
	if ($video.length) {
		$video.siblings('br').andSelf().remove();
		$target.css('width', 'auto');
		event.preventDefault();
		return;
	}
	if ($target.data('noembed'))
		return;
	var m = $target.attr('href').match(youtube_url_re);
	if (!m) {
		/* Shouldn't happen, but degrade to normal click action */
		return;
	}
	var start = 0;
	if (m[4]) {
		var t = m[4].match(youtube_time_re);
		if (t) {
			if (t[1])
				start += parseInt(t[1], 10) * 3600;
			if (t[2])
				start += parseInt(t[2], 10) * 60;
			if (t[3])
				start += parseInt(t[3], 10);
		}
	}

	var $obj = make_video(m[2], null, start);
	with_dom(function () {
		$target.css('width', video_dims().width).append('<br>', $obj);
	});
	event.preventDefault();
});

$(document).on('mouseenter', '.watch', function (event) {
	var $target = $(event.target);
	if ($target.data('requestedTitle'))
		return;
	$target.data('requestedTitle', true);
	/* Edit textNode in place so that we don't mess with the embed */
	var node = $target.contents().filter(function () {
		return this.nodeType === 3;
	})[0];
	if (!node)
		return;
	var orig = node.textContent;
	with_dom(function () {
		node.textContent = orig + '...';
	});
	var m = $target.attr('href').match(youtube_url_re);
	if (!m)
		return;

	$.ajax({
		url: '//gdata.youtube.com/feeds/api/videos/' + m[2],
		data: {v: '2', alt: 'jsonc'},
		dataType: 'json',
		success: function (data) {
			with_dom(gotInfo.bind(null, data));
		},
		error: function () {
			with_dom(function () {
				node.textContent = orig + '???';
			});
		},
	});

	function gotInfo(data) {
		var title = data && data.data && data.data.title;
		if (title) {
			node.textContent = orig + ': ' + title;
			$target.css({color: 'black'});
		}
		else
			node.textContent = orig + ' (gone?)';

		if (data && data.data && data.data.accessControl &&
				data.data.accessControl.embed == 'denied') {
			node.textContent += ' (EMBEDDING DISABLED)';
			$target.data('noembed', true);
		}
	}
});
