// remember which posts are mine for two days
var Mine = new Kioku('mine', 2);
// no cookie though
Mine.bake_cookie = function () { return false; };
$.cookie('mine', null); // TEMP

(function () {

var mine = Mine.read_all();

function extract_post_model(el) {
	/* incomplete */
	var info = {num: parseInt(el.id, 10)};
	var $el = $(el);
	/* TODO: do these all in one pass */
	var $header = $el.children('header');
	var $b = $header.find('b');
	if ($b.length)
		info.name = $b.text();
	var $code = $header.find('code');
	if ($code.length)
		info.trip = $code.text();
	var $time = $header.find('time');
	if ($time.length)
		info.time = new Date($time.attr('datetime')).getTime();

	var $fig = $el.children('figure');
	if ($fig.length){
		info.image = catchJSON($fig.data('img'));
		// These data attributes are only used for model extraction
		// Clean up for a prettier DOM
		$fig.removeAttr('data-img');
	}
	var $block = $el.children('blockquote');
	info.body = catchJSON($block.data('body'));
	$block.removeAttr('data-body');
	if (mine[info.num])
		info.mine = true;
	return info;
}

function extract_thread_model(section) {
	var replies = [];
	for (var i = 0; i < section.childElementCount; i++) {
		var el = section.children[i];
		if (el.tagName != 'ARTICLE')
			continue;
		var post = new Post(extract_post_model(el));
		new Article({model: post, el: el});
		// Add to all post collection
		Posts.add(post);
		replies.push(post);
	}
	var threadModel = extract_post_model(section),
		thread = new Thread(threadModel);
	Posts.add(threadModel);
	thread.set('replies', new Replies(replies));
	return thread;
}

function scan_threads_for_extraction() {
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

	if (THREAD)
		CurThread = Threads.get(THREAD);
}

scan_threads_for_extraction();
Mine.purge_expired_soon();

})();
