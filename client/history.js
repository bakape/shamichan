/*
 * Inter board/page/thread navigation with HTML5 history
 */

let main = require('./main'),
	{$, _, common, Extract, state} = main;

// Click handler for post/thread/board links
main.$doc.on ('click', 'a.history', function(event) {
	if (event.ctrlKey)
		return;
	readingSteiner(this.href, event, true);
});

// Loading status GIF
let $loading = $('#loadingImage');
main.comply('loading:show', () => $loading.show());
main.comply('loading:hide', () => $loading.hide());

// Navigate to the URL
function readingSteiner(url, event, needPush) {
	const nextState = state.read(url);
	// Does the link point to the same page as this one?
	if (_.isMatch(state.page.attributes, nextState))
		return;
	if (event)
		event.preventDefault();

	// Deal with hashes and query strings
	const split = url.split('#');
	let address = split[0] + (/\?/.test(split[0]) ? '&' : '?') + 'minimal=true';
	if (split.length !== 1)
		address += '#' + split[1];

	/*
	 * Fetch new DOM from the server
	 *
	 * Decided to go with a non-caching approach and instead relly on etags and
	 * CDN for HTML-only caching. This solution is already very fast on threads
	 * that are not several thousand posts large.
	 */
	$loading.show();
	let xhr = new XMLHttpRequest();
	xhr.open('GET', address);
	xhr.onload = function () {
		// In case the thread is dead, moderatator cookie expired or some
		// other shananigans
		if (this.status !== 200)
			return location.replace(this.url.split('?')[0]);

		main.trigger('state:clear');
		// Apply new DOM and load models
		main.$threads[0].innerHTML = this.response;
		// Set new page state
		state.page.set(nextState);
		// Reconfigure rendering singleton
		main.oneeSama.op = nextState.thread;
		new Extract();
		// Swap the database controller server-side
		main.command('send', [
			common.RESYNC,
			nextState.board,
			state.syncs,
			nextState.live
		]);

		if (needPush) {
			history.pushState(null, null, url);
			// Scroll to top on new pages with no hashes
			if (location.hash)
				main.command('scroll:aboveBanner');
			else
				window.scrollTo(0, 0);
		}
		$loading.hide();
	};
	xhr.send();
}

// For back and forward history events
window.onpopstate = function(event) {
	readingSteiner(event.target.location.href);
	main.command('scroll:aboveBanner');
};
