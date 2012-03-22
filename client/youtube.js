function make_video(id, params, dims, start) {
	if (!dims)
		dims = {width: 425, height: 355};
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
	var $obj = $('<object></object>').attr(dims);
	for (var name in params)
		$obj.append($('<param></param>').attr({name: name,
				value: params[name]}));
	$('<embed></embed>').attr(params).attr(dims).attr({src: uri,
		type: 'application/x-shockwave-flash'}).appendTo($obj);
	return $obj;
}

$(document).on('click', 'cite', function (event) {
	var $target = $(event.target);
	var m = $target.text().match(youtube_re);
	var start = 0;
	if (m[2]) {
		var t = m[2].match(youtube_time_re);
		if (t) {
			if (t[1])
				start += parseInt(t[1], 10) * 3600;
			if (t[2])
				start += parseInt(t[2], 10) * 60;
			if (t[3])
				start += parseInt(t[3], 10);
		}
	}
	var $obj = make_video(m[1], null, null, start);
	with_dom(function () {
		$target.replaceWith($obj);
	});
});
