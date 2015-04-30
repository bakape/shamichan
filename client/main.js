/*
 * Loads the depandancies in order and aggregates exports from various modules
 */

/*
 The entire bundle uses strict mode through the strictify browserify plugin
 */
"use strict";

/*
 * Because we are going to attach listeners to these all over the place, have
 * to load soome core modules in specific order. Also avoids nasty circular
 * dependaancy, by placing some of the exports here and not in child modules.
 */
var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone');

// Register jquery plugins
require('jquery.cookie');
// Bind jQuery to backbone
Backbone.$ = $;

/*
 * Since the language pack contains functions and we can not simply use those
 * with underscore templates, had to stringify those. Now we convert them back
 * to functions.
 */
exports.lang = window.lang;
['pluralize', 'capitalize', 'ago', 'abbrev_msg'].forEach(function(func) {
	eval('exports.lang[func] = ' + window.lang[func]);
});

/*
 These configs really should not be randomly toggled frequently. No need to put
 them in state.js, as they should not be hot-loaded. Anything that needs to be,
 can be moved to hot.js. Should prevent some bugs, but also reduce flexibility,
 for frequent togglers. Hmm.
 */
exports.config = window.config;

exports.isMobile = /Android|iP(?:hone|ad|od)|Windows Phone/
	.test(navigator.userAgent);
// Store them here, to avoid requiring modules in the wrong order
exports.send = function() {
};
exports.serverTimeOffset = 0;
exports.dispatcher = {};
exports.postForm = null;
exports.postModel = null;
// Read-only boards gets expanded later
exports.readOnly = ['archive'];

var state = require('./state'),
	common = require('../common');
exports.connSM = new common.FSM('load');
exports.postSM = new common.FSM('none');
state.page.set('tabID', common.random_id());

// Cached jQuery objects
exports.$doc = $(document);
exports.$threads = $('threads');
exports.$name = $('input[name=name]');
exports.$email = $('input[name=email]');

// Initialise main rendering object
var oneeSama = exports.oneeSama = new common.OneeSama(function(num) {
	// Core post link handler
	var frag;
	if (this.links && num in this.links) {
		var op = this.links[num],
			model = state.posts.get(num),
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
