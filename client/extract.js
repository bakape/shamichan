function extract_post_model(el) {
	/* incomplete */
	var info = {num: parseInt(el.id, 10)};
	var $article = $(el);
	/* TODO: do these all in one pass */
	var $header = $article.children('header');
	var $b = $header.find('b');
	if ($b.length)
		info.name = $b.text();
	var $code = $header.find('code');
	if ($code.length)
		info.trip = $code.text();
	var $time = $header.find('time');
	if ($time.length)
		info.time = new Date($time.attr('datetime')).getTime();

	var $fig = $article.children('figure');
	if ($fig.length) {
		var $cap = $fig.children('figcaption');
		var image = {
			MD5: $fig.data('md5'),
			src: $cap.children('a').text(),
		};

		/* guess for now */
		image.thumb = image.src;

		var m = $cap.find('i').text().match(
				/^\(\d+ \w+, (\d+)x(\d+),/);
		if (m)
			image.dims = [parseInt(m[1], 10), parseInt(m[2], 10)];
		info.image = image;
	}
	return new Post(info);
}

function extract_thread_model(section) {
	var replies = [];
	for (var i = 0; i < section.childElementCount; i++) {
		var el = section.children[i];
		if (el.tagName != 'ARTICLE')
			continue;
		var post = extract_post_model(el);
		new Article({model: post, el: el});
		replies.push(post);
	}
	return new Thread({
		num: parseInt(section.id, 10),
		replies: new Replies(replies),
	});
}

(function scan_threads_for_extraction() {
	var bod = document.body;
	var threads = [];
	for (var i = 0; i < bod.childElementCount; i++) {
		var el = bod.children[i];
		if (el.tagName != 'SECTION')
			continue;
		var thread = extract_thread_model(el);
		new Section({model: thread, el: el});
		threads.push(thread);
	}
	Threads.add(threads);
})();
