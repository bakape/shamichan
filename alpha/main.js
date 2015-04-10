/*
 * Loads the depandancies in order and aggregates exports from various modules
 */

/*
 * Because we are going to attach listeners to these all over the place, have
 * to load soome core modules in specific order. Also avoids nasty circular
 * dependaancy, by placing some of the exports here and not in child modules.
 */
var $ = require('jquery'),
	Backbone = require('backbone'),
	common = require('../common');

// Register jquery plugins
require('jquery.cookie');
// Bind jQuery to backbone
Backbone.$ = $;

exports.isMobile = /Android|iP(?:hone|ad|od)|Windows Phone/
	.test(navigator.userAgent);

// Store them here, to avoid requiring modules in the wrong order
exports.send = function() {};
exports.dispatcher = {};
exports.syncs = {};
exports.connSM = new common.FSM('load');
exports.postSM = new common.FSM('none');

var PostCollection = exports.PostCollection = Backbone.Collection.extend({
	idAttribute: 'num'
});
// All posts currently displayed
var posts = exports.posts = new PostCollection();
/*
* All threads currently displayed. Threads are also posts, so they are in
* both collections. This seperation is needed, not to search through all
* posts, to find a thread.
*/
exports.threads = new PostCollection();

// Cached jQuery objects
exports.$doc = $(document);
exports.$threads = $('threads');

var state = require('./state');

// Initialise main rendering object
var oneeSama = exports.oneeSama = new common.OneeSama(function(num) {
	// Core post link handler
	var frag;
	if (this.links && num in this.links) {
		var op = this.links[num];
		// FIXME: Threads not done yet
		var model = posts.get(num),
			desc = model && model.get('mine') && '(You)';
		frag = this.post_ref(num, op, desc);
	}
	else
		frag = '>>' + num;
	this.callback(frag);
});
oneeSama.full = oneeSama.op = state.page.get('thread');

// The require chain also loads some core dependancies
var Extract = require('./extract');
new Extract();

/*
 * Load this after the extraction, so models are already populated, when the we
 * attach listeners to the various cross-thread links. If the user clicks a link
 * before that, standard href's will direct him. This also adds some SEO.
 */
require('./history');

// Start the client
require('./client');
