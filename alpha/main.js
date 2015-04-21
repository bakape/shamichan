/*
 * Loads the depandancies in order and aggregates exports from various modules
 */

/*
 * Because we are going to attach listeners to these all over the place, have
 * to load soome core modules in specific order. Also avoids nasty circular
 * dependaancy, by placing some of the exports here and not in child modules.
 */
var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone');

/*
 * Since the language pack contains functions and we can not simply use those
 * with underscore templates, had to stringify those. Now we convert them back
 * to functions.
 */
exports.lang = window.lang;
['pluralize', 'capitalize', 'ago', 'abbrev_msg'].forEach(function(func) {
	eval('exports.lang[func] = ' + window.lang[func]);
});

var common = require('../common');

// Register jquery plugins
require('jquery.cookie');
// Bind jQuery to backbone
Backbone.$ = $;

exports.isMobile = /Android|iP(?:hone|ad|od)|Windows Phone/
	.test(navigator.userAgent);

// Store them here, to avoid requiring modules in the wrong order
exports.send = function() {};
exports.serverTimeOffset = 0;
exports.dispatcher = {};
exports.connSM = new common.FSM('load');
exports.postSM = new common.FSM('none');

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
		var model = state.posts.get(num),
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

// Start the client
require('./client');

// Load auxilary modules
require('./history');