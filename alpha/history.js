/*
 * Inter board/page/thread navigation with HTML5 history
 */

var $ = require('jquery'),
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
	// Deal with hashes and query strings
	var split = this.url.split('#'),
		url = split[0] + (/\?/.test(split[0]) ? '&' : '?') + 'minimal=true';
	if (split.length !== 1)
		url += '#' + split[1];

	/*
	 * Fetch new DOM from the server
	 *
	 * Decided to go with a non-caching approach and instead relly on etags and
	 * CDN for HTML-only caching. This solution is already very fast on threads
	 * that are not several thousand posts large.
	 */
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
	history.pushState(null, null, this.nextState.href);
};

// For back and forward history events
window.onpopstate = function(event) {
	var rs = new ReadingSteiner(event.target.location.href);
	// Also protects against [Top] and [Bottom] triggers
	if (!rs.check())
		return;
	rs.navigate();
};