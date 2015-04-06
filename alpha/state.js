/*
 * Central model keeping the state of the page
 */
var Backbone = require('backbone');

// Read initial page state from URL
var path = location.pathname,
	thread = path.match(/\/(\d+)$/),
	page = path.match(/\/page(\d+)$/);
exports.page = new Backbone.Model({
	board: path.match(/^\/(.+?)\//)[1],
	thread: thread ? parseInt(thread[1], 10) : 0,
	page: page ? parseInt(page[1], 10) : -1
});

/*
 * Not sure how many of these are going to be  more useful than a property of
 * the window object. We'll as we go, I guess.
 */
['config', 'imagerConfig', 'reportConfig', 'hotConfig'].forEach(function(type) {
	exports[type] = new Backbone.Model(window[type]);
});
