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
// Bind Backbone.Radio
var radio = require('backbone.radio');

// Central aplication object and message bus
let main = module.exports = radio.channel('main')

/*
 * Since the language pack contains functions and we can not simply use those
 * with underscore templates, had to stringify those. Now we convert them back
 * to functions.
 */
main.lang = window.lang;
['pluralize', 'capitalize', 'ago', 'abbrev_msg'].forEach(function(func) {
	eval('main.lang[func] = ' + main.lang[func]);
});

/*
 These configs really should not be randomly toggled frequently. No need to put
 them in state.js, as they should not be hot-loaded. Anything that needs to be,
 can be moved to hot.js. Should prevent some bugs, but also reduce flexibility,
 for frequent togglers. Hmm.
 */
main.config = window.config;


if (main.config.DEBUG) {
	// Export Backbone instance for easier debugging
	window.Backbone = Backbone;
	// Log all channel traffic
	radio.DEBUG = true;
	radio.tuneIn('main');
}


main.isMobile = /Android|iP(?:hone|ad|od)|Windows Phone/
	.test(navigator.userAgent);
// Store them here, to avoid requiring modules in the wrong order
main.send = function() {};
main.serverTimeOffset = 0;
main.dispatcher = {};
main.postForm = null;
main.postModel = null;
main.openPostBox = function() {};
// Read-only boards gets expanded later
main.readOnly = ['archive'];

/*
 Core modules. The other will be more or less decoupled, but these are the
 monolithic foundation.
 */
let state = main.state = require('./state');
let	common = main.common = require('../common');
// Initialise main rendering object
var oneeSama = main.oneeSama = new common.OneeSama(function(num) {
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
main.options = require('./options');

main.connSM = new common.FSM('load');
main.postSM = new common.FSM('none');
state.page.set('tabID', common.random_id());

// Cached jQuery objects
main.$doc = $(document);
main.$threads = $('threads');
main.$name = $('input[name=name]');
main.$email = $('input[name=email]');



// The require chain also loads some core dependancies
var Extract = require('./extract');
new Extract();

main.banner = require('./banner');
main.background = require('./options/background');

// Start the client
require('./client');

// Load auxilary modules
require('./history');
require('./hover');
