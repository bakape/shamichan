/*
 * Inter board/page/thread navigation with HTML5 history
 */

let main = require('./main'),
	{$, _, common, Extract, state} = main;

// Click handler for post/thread/board links
main.$doc.on ('click', 'a.history', function(event) {
	if (event.ctrlKey)
		return;
	readingSteiner(this.href, event);
});

// Loading status GIF
let $loading = $('#loadingImage');
main.reply('loading:show', () => $loading.show());
main.reply('loading:hide', () => $loading.hide());

// Navigate to the URL
function readingSteiner(url, event) {
	let nextState = state.read(url);
	// Does the link point to the same page as this one?
	if (_.isMatch(state.page.attributes, nextState))
		return;

	// Disconnect server-side Yakusoku in preparation for navigating away.
	// This helps avoid duplicate messages mid-navigation.
	main.request('connection:lock');

	if (event)
		event.preventDefault();

	// Deal with hashes and query strings
	const split = url.split('#');
	let address = split[0] + (/\?/.test(split[0]) ? '&' : '?') + 'minimal=true';
	if (split.length !== 1)
		address += '#' + split[1];

	/*
	 * Fetch new DOM from the server
	 * Decided to go withthout dedicated caching and use etags for browser
	 * cache verification.
	 */
	$loading.show();
	const xhr = new XMLHttpRequest();
	xhr.open('GET', address);
	xhr.onload = function () {
		// In case the thread is dead, moderatator cookie expired or some
		// other shenanigans
		if (this.status !== 200) {
			$loading.hide()
			return alert(this.status)
		}

		// Was redirected to different thread/board
		if (baseURL(url) !== baseURL(this.responseURL))
			nextState = state.read(this.responseURL);
		main.request('postSM:feed', 'done');
		main.trigger('state:clear');

		// Apply new DOM and load models
		main.$threads[0].innerHTML = this.response;

		// Set new page state
		state.page.set(nextState);

		// Reconfigure rendering singleton
		main.oneeSama.op = nextState.thread;
		new Extract(nextState.catalog);

		// Swap the database controller server-side. Catalog does not use a
		// Yakusoku(), so not needed.
		if (!nextState.catalog) {
			main.request('connection:unlock', [common.RESYNC, nextState.board,
				state.syncs, nextState.live]);
		}

		if (event) {
			history.pushState(null, null, nextState.href);

			// Scroll to top on new pages with no hashes
			if (location.hash)
				main.request('scroll:aboveBanner');
			else
				window.scrollTo(0, 0);
		}
		$loading.hide();
	};
	xhr.send();
}

function baseURL(url) {
	return url.split(/[\?#]/)[0];
}

// For back and forward history events
window.onpopstate = function(event) {
	readingSteiner(event.target.location.href);
	main.request('scroll:aboveBanner');
};
