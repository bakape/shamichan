/*
 * Central model keeping the state of the page
 */

var Backbone = require('backbone'),
	common = require('../common'),
	main = require('./main');

// Read page state by parsing a URL
var read = exports.read = function(url) {
	const href = url.split('#')[0],
		thread = href.match(/\/(\d+)(:?#\d+)?(?:[\?&]\w+=\w+)*$/),
		page = href.match(/\/page(\d+)$/),
		// Display last N posts setting on thread pages
		lastN = href.match(/[\?&]last=(\d+)/);
	return {
		href: href,
		board: href.match(/\/([a-zA-Z0-9]+?)\//)[1],
		thread: thread ? parseInt(thread[1], 10) : 0,
		page: page ? parseInt(page[1], 10) : -1,
		lastN: lastN ? parseInt(lastN[1], 10) : 0,
		/*
		 * Indicates if on the 'live' board page, which needs extra server-side
		 * logic.
		 */
		live: page == -1 && thread == 0
	};
};

// Initial page state
var page = exports.page = new Backbone.Model(read(location.href));

/*
 * Not sure how many of these are going to be  more useful than a property of
 * the window object. We'll as we go, I guess.
 */
['config', 'imagerConfig', 'reportConfig', 'hotConfig'].forEach(function(type) {
	exports[type] = new Backbone.Model(window[type]);
});
// Hash of all the config variables
exports.configHash = window.configHash;

var PostCollection = Backbone.Collection.extend({
	idAttribute: 'num'
});
// All posts currently displayed
var posts = exports.posts = new PostCollection();
/*
* All threads currently displayed. Threads are also posts.
* This seperation is needed, not to search through all posts, to find a thread.
*/
var threads = exports.threads = new Set();

exports.getThread = function(num) {
	if (!threads.has(num))
		return null;
	return posts.get(num);
};

// Tracks the synchronisation counter of each thread
exports.syncs = {};

// Clear current post state, DOM and server synchronisation and apply the new
exports.replace = function(newState, render) {
	/*
	 * Emptying the whole element should be faster than removing each post
	 * individually through models and listeners. Not that the `remove()`s
	 * don't fire anymore...
	 */
	main.$threads.empty();
	threads.clear();
	posts.models.forEach(function(model) {
		model.destroy();		
	});
	// Prevent old threads from syncing
	exports.syncs = {};
	// Set new page state
	// TODO: Reload board-specific options on change
	page.set(newState);
	// Rendering and extraction as needed
	render();

	// Swap the database controller server-side
	main.send([
		common.RESYNC,
		page.get('board'),
		exports.syncs,
		page.get('live')
	]);
};