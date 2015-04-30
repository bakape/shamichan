/*
 * Central model keeping the state of the page
 */

var $ = require('jquery'),
	Backbone = require('backbone'),
	memory = require('./memory');

// Read page state by parsing a URL
var read = exports.read = function(url) {
	const href = url.split('#')[0],
		// Display last N posts setting on thread pages
		lastN = href.match(/[\?&]last=(\d+)/);
	var thread = href.match(/\/(\d+)(:?#\d+)?(?:[\?&]\w+=\w+)*$/),
		page = href.match(/\/page(\d+)$/);
	thread = thread ? parseInt(thread[1], 10) : 0;
	page = page ? parseInt(page[1], 10) : -1;
	return {
		href,
		board: href.match(/\/([a-zA-Z0-9]+?)\//)[1],
		thread,
		page,
		lastN: lastN ? parseInt(lastN[1], 10) : 0,
		/*
		 * Indicates if on the 'live' board page, which needs extra server-side
		 * logic.
		 */
		live: page === -1 && thread === 0
	};
};

// Initial page state
var page = exports.page = new Backbone.Model(read(location.href));

// Hot-reloadable configuration
// TODO: We need actual listeners to this model for hot reloads.
exports.hotConfig = new Backbone.Model(window.hotConfig);
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
// Posts I made in this tab
exports.ownPosts = {};
// remember which posts are mine for two days
exports.mine = new memory('mine', 2);
// no cookie though
exports.mine.bake_cookie = function () { return false; };
$.cookie('mine', null); // TEMP
