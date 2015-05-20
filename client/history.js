/*
 * Inter board/page/thread navigation with HTML5 history
 */

var $ = require('jquery'),
	_ = require('underscore'),
	common = require('../common'),
	Extract = require('./extract'),
	main = require('./main'),
	state = require('./state');

// Click handler for post/thread/board links
main.$doc.on ('click', 'a.history', function(event) {
	readingSteiner(this.href, event, true);
});

// Navigate to the URL
function readingSteiner(url, event, needPush) {
	const nextState = state.read(url);
	// Does the link point to the same page as this one?
	if (_.isMatch(state.page.attributes, nextState))
		return;
	if (event)
		event.preventDefault();

	// Deal with hashes and query strings
	var split = url.split('#'),
		address = split[0] + (/\?/.test(split[0]) ? '&' : '?') + 'minimal=true';
	if (split.length !== 1)
		address += '#' + split[1];

	/*
	 * Fetch new DOM from the server
	 *
	 * Decided to go with a non-caching approach and instead relly on etags and
	 * CDN for HTML-only caching. This solution is already very fast on threads
	 * that are not several thousand posts large.
	 */
	var $loading = $('#loadingImage').show();
	$.get(address, function(data) {
		if (!data)
			return alert('Fetch failed: ' + url);

		/*
		 * Emptying the whole element should be faster than removing each post
		 * individually through models and listeners. Not that the `remove()`s
		 * don't fire anymore...
		 */
		main.$threads.empty();
		state.posts.models.forEach(function(model) {
			model.remove();
		});
		// Prevent old threads from syncing
		state.syncs = {};
		// Apply new DOM and load models
		main.$threads.html(data);
		// Set new page state
		state.page.set(nextState);
		// Reconfigure rendering singleton
		main.oneeSama.full = main.oneeSama.op = nextState.thread;
		main.command('massExpander:unset');
		new Extract();
		// Swap the database controller server-side
		main.command('send', [
			common.RESYNC,
			state.page.get('board'),
			state.syncs,
			state.page.get('live')
		]);

		if (needPush){
			history.pushState(null, null, url);
			// Scroll to top on new pages with no hashes
			if (!location.hash)
				window.scrollTo(0, 0);
			else
				main.command('scroll:aboveBanner');
		}
		$loading.hide();
	});
}

// For back and forward history events
window.onpopstate = function(event) {
	readingSteiner(event.target.location.href);
	main.command('scroll:aboveBanner');
};
