/*
 * Inter board/page/thread navigation with HTML5 history
 */

var $ = require('jquery'),
	conn = require('./connection'),
	Extract = require('./extract'),
	main = require('./main'),
	state = require('./state');

// Click handler for post/thread/board links
main.$doc.on ('click', 'a.history', function(event) {
	var rs = new ReadingSteiner(this.href);
	if (!rs.check())
		return;
	event.preventDefault();
	rs.navigate();
	rs.push();
});

function ReadingSteiner(url, needPush) {
	this.url = url;
	this.nextState = state.read(url);
	this.needPush = needPush;
}

var RS = ReadingSteiner.prototype;

// Does the link point to the same page as this one?
RS.check = function() {
	return JSON.stringify(this.nextState)
		!= JSON.stringify(state.page.attributes);
};

// Go to the URL
RS.navigate = function(cb) {
	cache();
	/*
	 * Board pages are too dynamic. Caching those will actually create overhead
	 * as the client syncs through redis -> websocket -> client-side rendering.
	 * Best just fetch them prerendered from the server each time.
	 */
	if (!this.nextState.thread)
		return this.fetch(cb);
	const key = storageKey(this.nextState),
		html = sessionStorage[key + 'html'],
		syncs = sessionStorage[key + 'syncs'];
	var posts = sessionStorage[key + 'posts'],
		threads = sessionStorage[key + 'threads'];
	// Verify all keys exist. Deals with interuptions mid-cache.
	//if (!html || !syncs || !posts || !threads)
		return this.fetch(cb);
	this.load(html, posts, threads);
};

// Dump page state to sessionStorage
function cache() {
	const page = state.page.attributes;
	// Not a thread
	if (!page.thread)
		return;

	const key = storageKey(page);
	sessionStorage[key + 'html'] = main.$threads.html();
	sessionStorage[key + 'syncs'] = JSON.stringify(state.syncs);
	sessionStorage[key + 'posts'] = JSON.stringify(state.posts);

	// Because it's a Set()
	var threads = [];
	state.threads.forEach(function(thread) {
		threads.push(thread);		
	});
	sessionStorage[key + 'threads'] = JSON.stringify(threads);
}

// Generate key name
function storageKey(page) {
	return 'thread:' + page.thread + ':' + page.lastN + ':';
}

// Fetch new DOM from the server
RS.fetch = function(cb) {
	// Deal with hashes and query strings
	var split = this.url.split('#'),
		url = split[0] + (/\?/.test(split[0]) ? '&' : '?') + 'minimal=true';
	if (split.length !== 1)
		url += '#' + split[1];

	var self = this;
	$.get(url, function(data) {
		if (!data)
			return alert('Fetch failed: ' + url);

		// Apply new state and DOM
		state.replace(self.nextState, function() {
			main.$threads.html(data);
			new Extract();
		});
		if (self.needPush)
			self.push();
	});
};

RS.push = function() {
	history.pushState(null, null, this.nextState.href)
};

// For back and forward history events
window.onpopstate = function(event) {
	var rs = new ReadingSteiner(event.target.location.href);
	// Also protects against [Top] and [Bottom] triggers
	if (!rs.check())
		return;
	rs.navigate();
};