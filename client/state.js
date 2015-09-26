/*
 * Central model keeping the state of the page
 */

let main = require('./main'),
	{_, Backbone} = main;

// Read page state by parsing a URL
function read(url) {
	// Strip minimal mode, so we save a proper URL into History
	const href = url.split('#')[0].replace(/[\?&]minimal=true/, '');

	// Display last N posts setting on thread pages
	let lastN = href.match(/[\?&]last=(\d+)/),
		thread = href.match(/\/(\d+)(:?#\d+)?(?:[\?&]\w+=\w+)*$/),
		page = href.match(/\/page(\d+)$/);
	lastN = lastN ? parseInt(lastN[1], 10) : 0;
	thread = thread ? parseInt(thread[1], 10) : 0;
	page = page ? parseInt(page[1], 10) : -1;
	return {
		href,
		thread,
		page,
		lastN,
		board: href.match(/\/([a-zA-Z0-9]+?)\//)[1],
		catalog: /\/catalog/.test(href),

		// Indicates if on the 'live' board page, which needs extra server-side
		// logic.
		live: page === -1 && thread === 0
	};
}
exports.read = read;

// Initial page state
let page = exports.page = new Backbone.Model(read(location.href));

// Hot-reloadable configuration
// TODO: We need actual listeners to this model for hot reloads.
exports.hotConfig = new Backbone.Model(imports.hotConfig);
// Hash of all the config variables
exports.configHash = imports.configHash;

// Tracks the synchronisation counter of each thread
exports.syncs = {};
// Posts I made in this tab
exports.ownPosts = {};
// remember which posts are mine for two days
let mine = exports.mine = new main.Memory('mine', 2, true);

// All posts currently displayed
let PostCollection = Backbone.Collection.extend({
	// Needed, because we use different model classes in the same
	// collection. Apperently Backbone >=1.2.0 no longer picks those up
	// automatically.
	modelId(attrs) {
		return attrs.num;
	}
});
let posts = exports.posts = new PostCollection();
main.on('state:clear', function() {
	/*
	 * Emptying the whole element should be faster than removing each post
	 * individually through models and listeners. Not that the `remove()`s
	 * don't fire anymore...
	 */
	main.$threads[0].innerHTML = '';
	const models = posts.models;
	for (let i = 0, l = models.length; i < l; i++) {
		// The <threads> tag has already been emptied, no need to perform
		// element removal with the default `.remove()` method
		models[i].dispatch('stopListening');
	}
	posts.reset();
	// Prevent old threads from syncing
	exports.syncs = {};
	main.request('massExpander:unset');
});

// Post links verified server-side
let links = exports.links = {};

function addLinks(addition) {
	if (addition) {
		_.extend(links, addition);
	}
}
exports.addLinks = addLinks;
