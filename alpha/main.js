/*
 * Loads the depandancies in order and aggregates exports from various modules
 */

/*
 * Because we are going to attach listeners to these all over the place, have
 * to load soome core modules in specific order.
 */
var $ = require('jquery'),
	Backbone = require('backbone'),
	state = require('./state'),
	common = require('../common'),
	lang = require('../lang/');

// Register jquery plugins
require('jquery.cookie');
// Bind jQuery to backbone
Backbone.$ = $;

// Core post link handler
function tamashi(num) {
	var frag;
	if (this.links && num in this.links) {
		var op = this.links[num];
		// FIXME: Threads not done yet
		var post = Threads.lookup(num, op);
		var desc = post && post.get('mine') && '(You)';
		frag = this.post_ref(num, op, desc);
	}
	else
		frag = '>>' + num;
	this.callback(frag);
}

// Initialise main rendering object
var oneeSama = exports.oneeSama = new common.OneeSama(tamashi);
oneeSama.full = oneeSama.op = state.page.get('thread');

// Read language preference
// TODO: Remove, when options panel rendering is moved to server/state.js
const langSet = state.config.get('LANGS')[$.cookie('lang')];
if (langSet)
	oneeSama.lang = lang[langSet];

const isMobile = exports.isMobile = /Android|iP(?:hone|ad|od)|Windows Phone/
	.test(navigator.userAgent);

// Store them here, to avoid requiring modules in the wrong order
exports.send = function() {};
exports.dispatcher = {};
exports.syncs = {};
exports.connSM = new common.FSM('load');
exports.postSM = new common.FSM('none');

// Cached jQuery objects
exports.$doc = $(document);
exports.$threads = $('threads');

// The require chain also loads some core dependancies
var Extract = require('./extract');
new Extract();