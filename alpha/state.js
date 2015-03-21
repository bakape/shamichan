/*
 * Central model keeping the state of the page
 */
var	Backbone = require('backbone');

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
 * Because we are going to attach listeners to these all over the place, have to
 * be loaded first. The order seems pretty solid
 */
var c = ['config', 'imagerConfig', 'reportConfig', 'hotconfig'],
	type;
for (var i = 0; i < c.length; i++) {
	type = c[i];
	exports[type] = new Backbone.Model(window[type]);
}