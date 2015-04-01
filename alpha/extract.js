/*
 * Extact model data from the thread tree HTML and populate models and views
 */

var $ = require('jquery'),
	main = require('./main'),
	memory = require('./memory'),
	posts = require('./posts/');

// remember which posts are mine for two days
var Mine = new memory('mine', 2);
// no cookie though
Mine.bake_cookie = function () { return false; };
$.cookie('mine', null); // TEMP

function Extract() {
	this.mine = Mine.read_all();
	var self = this;
	main.$threads.children('section').each(function() {
		self.extractThread($(this));
	});
}
module.exports = Extract;

Extract.prototype.extractThread = function($section) {
	var replies = [],
		self = this;
	$section.children('article').each(function() {
		var post = new posts.PostModel(self.extractModel($(this)));
		new posts.Article({
			model: post,
			el: this
		});
		replies.push(post);
		// Add to displayed post collection
		posts.posts.add(post);
	});
	// Extract the model of the OP
	var threadModel = new posts.ThreadModel(this.extractModel($section));
	// Add all replies to the threads reply collection
	threadModel.replies.add(replies);
	// Add to both collections, for less expensive searches
	posts.threads.add(threadModel);
	posts.posts.add(threadModel);
	new posts.Section({
		model: threadModel,
		el: $section[0]
	});
};

Extract.prototype.extractModel = function($el) {
	var info = {num: parseInt($el.attr('id'), 10)};
	var $header = $el.children('header'),
		$b = $header.find('b');
	if ($b.length)
		info.name = $b.text();
	var $code = $header.find('code');
	if ($code.length)
		info.trip = $code.text();
	var $time = $header.find('time');
	if ($time.length)
		info.time = new Date($time.attr('datetime')).getTime();

	var $fig = $el.children('figure');
	if ($fig.length)
		info.image = catchJSON($fig.data('img'));
	info.body = catchJSON($el.children('blockquote').data('body'));
	if (this.mine[info.num])
		info.mine = true;
	return info;
};

function catchJSON(string) {
	return JSON.parse(decodeURIComponent(string));
}
