/*
 * Loads the depandancies in order and aggregates exports from various modules
 */

/*
 * Because we are going to attach listeners to these all over the place, have
 * to be loaded first. The order seems pretty solid
 */

var $ = require('jquery'),
	Backbone = require('backbone'),
	state = require('./state'),
	common = require('../common');

// Register jquery plugins
require('jquery.cookie');

// Bind jQuery to backbone
Backbone.$ = $;

var isMobile = exports.isMobile = /Android|iP(?:hone|ad|od)|Windows Phone/.test(
	navigator.userAgent);
var oneeSama = exports.oneeSama = new common.OneeSama(function(num) {
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
});
oneeSama.full = oneeSama.op = state.page.get('thread');

exports.send = function() {};
exports.dispatcher = {};

var options = require('./options'),
	models = require('./models'),
	extract = require('./extract');


//extract.extract_threads();