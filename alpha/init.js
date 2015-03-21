/*
 * Loads the top priority dependancies
 */

/*
 * Because we are going to attach listeners to these all over the place, have
 * to be loaded first. The order seems pretty solid
 */

var state = require('./state'),
	common = require('../common');

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

// Later gets overwrittten in conn.js
exports.send = function() {};

var options = require('./options'),
	models = require('./models'),
	extract = require('./extract');


//extract.extract_threads();