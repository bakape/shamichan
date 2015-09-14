/*
Youtube, soundcloud and pastebin link embeding
 */

// TODO: DRY this shit

let main = require('../main'),
	{$} = main;

// >80 char rule
const youtube_url_re = exports.youtube_url_re = /(?:>>>*?)?(?:https?:\/\/)?(?:www\.|m.)?youtube\.com\/watch\/?\?((?:[^\s#&=]+=[^\s#&]*&)*)?v=([\w-]{11})((?:&[^\s#&=]+=[^\s#&]*)*)&?(#t=[\dhms]{1,9})?/,
	youtube_short_re = exports.youtube_short_re = /(?:>>>*?)?(?:https?:\/\/)?(?:www\.|m.)?youtu.be\/([\w-]{11})\??(t=[\dhms]{1,9})?/,
	youtube_time_re = exports.youtube_time_re = /^#t=(?:(\d\d?)h)?(?:(\d{1,3})m)?(?:(\d{1,3})s)?$/,
	youtube_short_time_re = exports.youtube_short_time_re = /^t=(?:(\d\d?)h)?(?:(\d{1,3})m)?(?:(\d{1,3})s)?$/;

function make_video(id, params, start) {
	if (!params)
		params = {allowFullScreen: 'true'};
	params.allowScriptAccess = 'always';
	var query = {
		autohide: 1,
		fs: 1,
		modestbranding: 1,
		origin: document.location.origin,
		rel: 0,
		showinfo: 0
	};
	if (start)
		query.start = start;
	if (params.autoplay)
		query.autoplay = params.autoplay;
	if (params.loop) {
		query.loop = '1';
		query.playlist = id;
	}

	return $('<iframe></iframe>', {
		type: 'text/html',
		src: encodeURI('https://www.youtube.com/embed/' + id) + '?'
			+ $.param(query),
		frameborder: '0',
		attr: video_dims(),
		class: 'youtube-player'
	});
}

function video_dims() {
	if (window.screen && screen.width <= 320)
		return {width: 250, height: 150};
	else
		return {width: 560, height: 340};
}

main.$threads.on('click', '.watch', function(e) {
	if (e.which > 1 || e.metaKey || e.ctrlKey || e.altKey || e.shiftKey)
		return;
	var $target = $(e.target);

	// maybe squash that double-play bug? ugh, really
	if (!$target.is('a'))
		return;

	var $video = $target.find('iframe');
	if ($video.length) {
		$video.siblings('br').andSelf().remove();
		$target.css('width', 'auto');
		return false;
	}
	if ($target.data('noembed'))
		return;
	//check if longURL, if that fails, check if shortURL
	var m = $target.attr('href').match(youtube_url_re);
	if (!m) {
		m = $target.attr('href').match(youtube_short_re);
		if (!m)
		// Shouldn't happen, but degrade to normal click action
			return;
		timeCall(m[1],m[2],youtube_short_time_re);
	}
	timeCall(m[2],m[4],youtube_time_re);
	function timeCall(url, time, timeRex){
		var start = 0;
		if (time){
			var t = time.match(timeRex);
			if (t) {
				if (t[1])
					start += parseInt(t[1], 10) * 3600;
				if (t[2])
					start += parseInt(t[2], 10) * 60;
				if (t[3])
					start += parseInt(t[3], 10);
			}
		}
		main.follow(() =>
			$target
				.css('width', video_dims().width)
				.append('<br>', make_video(url, null, start))
		);
	}
	return false;
});

main.$threads.on('mouseenter', '.watch', function (event) {
	var $target = $(event.target);
	if ($target.data('requestedTitle'))
		return;
	$target.data('requestedTitle', true);
	// Edit textNode in place so that we don't mess with the embed
	var node = $target.contents().filter(function () {
		return this.nodeType === 3;
	})[0];
	if (!node)
		return;
	const orig = node.textContent;
	main.follow(() => node.textContent = orig + '...');
	var m = $target.attr('href').match(youtube_url_re);
	if (!m){
		m = $target.attr('href').match(youtube_short_re);
		if(!m)
			return;
		$.ajax({
			url: '//gdata.youtube.com/feeds/api/videos/' + m[1],
			data: {v: '2', alt: 'jsonc'},
			dataType: 'json',
			success: (data) =>
				main.follow(() => gotInfo.bind(null, data)),
			error: () =>
				main.follow(() => node.textContent = orig + '???')
		});
	}

	$.ajax({
		url: '//gdata.youtube.com/feeds/api/videos/' + m[2],
		data: {v: '2', alt: 'jsonc'},
		dataType: 'json',
		success: data =>
			main.follow(() => gotInfo.bind(null, data)),
		error: () =>
			main.follow(() => node.textContent = orig + '???')
	});
	// Creates the Titles upon hover
	// NOTE: Condense gotInfos into single function
	function gotInfo(data) {
		var title = data && data.data && data.data.title;
		if (title) {
			node.textContent = orig + ': ' + title;
			$target.css({color: 'black'});
		}
		else
			node.textContent = orig + ' (gone?)';

		if (data
			&& data.data
			&& data.data.accessControl
			&& data.data.accessControl.embed === 'denied'
		) {
			node.textContent += ' (EMBEDDING DISABLED)';
			$target.data('noembed', true);
		}
	}
});

function make_embed(uri, params, dims) {
	var $obj = $('<object/>', {attr: dims});
	for (var name in params) {
		$('<param/>', {
			attr: {
				name: name,
				value: params[name]
			}
		}).appendTo($obj);
	}
	$('<embed/>', {
		src: uri,
		type: 'application/x-shockwave-flash'
	})
		.attr(dims)
		.attr(params)
		.appendTo($obj);
	return $obj;
}

/* SOUNDCLOUD */

const soundcloud_url_re = exports.soundcloud_url_re = /(?:>>>*?)?(?:https?:\/\/)?(?:www\.)?soundcloud\.com\/([\w-]{1,40}\/[\w-]{1,80})\/?/;

function make_soundcloud(path, dims) {
	var uri = 'https://player.soundcloud.com/player.swf?'
		+ $.param({url: 'http://soundcloud.com/' + path});
	return make_embed(uri, {movie: uri}, dims);
}

main.$threads.on('click', '.soundcloud', function (e) {
	if (e.which > 1 || e.ctrlKey || e.altKey || e.shiftKey || e.metaKey)
		return;
	var $target = $(e.target);

	var $obj = $target.find('object');
	if ($obj.length) {
		$obj.siblings('br').andSelf().remove();
		$target.css('width', 'auto');
		return false;
	}
	var m = $target.attr('href').match(soundcloud_url_re);
	if (!m)
		// Shouldn't happen, but degrade to normal click action
		return;
	const width = Math.round($(window).innerWidth() * 0.75);
	$obj = make_soundcloud(m[1], {width: width, height: 81});
	main.follow(() => $target.css('width', width).append('<br>', $obj));
	return false;
});

// lol copy pasta
main.$threads.on('mouseenter', '.soundcloud', function (event) {
	var $target = $(event.target);
	if ($target.data('requestedTitle'))
		return;
	$target.data('requestedTitle', true);
	// Edit textNode in place so that we don't mess with the embed
	var node = $target.contents().filter(function () {
		return this.nodeType === 3;
	})[0];
	if (!node)
		return;
	var orig = node.textContent;
	main.follow(() => node.textContent = orig + '...');
	var m = $target.attr('href').match(soundcloud_url_re);
	if (!m)
		return;

	$.ajax({
		url: '//soundcloud.com/oembed',
		data: {format: 'json', url: 'http://soundcloud.com/' + m[1]},
		dataType: 'json',
		success: data =>
			main.follow(() => gotInfo.bind(null, data)),
		error: () =>
			main.follow(() => node.textContent = orig + '???')
	});

	function gotInfo(data) {
		var title = data && data.title;
		if (title) {
			node.textContent = orig + ': ' + title;
			$target.css({color: 'black'});
		}
		else
			node.textContent = orig + ' (gone?)';
	}
});

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
	main.request('scroll:follow', () =>
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
	);
	return false;
});
