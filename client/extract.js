/*
 * Extact model data from the thread tree HTML and populate models and views
 */

var $ = require('jquery'),
	main = require('./main'),
	state = require('./state'),
	posts = require('./posts');

var Extract = module.exports = function() {
	this.mine = state.mine.read_all();
	this.json = JSON.parse(main.$threads.children('#postData').text());
	var self = this;
	main.$threads.children('section').each(function() {
		self.extractThread($(this));
	});
};

Extract.prototype.extractThread = function($section) {
	var self = this;
	$section.children('article').each(function() {
		new posts.Article({
			model: new posts.models.Post(self.extractModel(this)),
			el: this
		});
	});
	// Extract the model of the OP
	var model = this.extractModel($section[0]);
	new posts.Section({
		model: new posts.models.Thread(model),
		el: $section[0]
	});
	/*
	 * Read the sync ID of the thread. Used later for syncronising with the
	 * server.
	 */
	state.syncs[model.num] = parseInt(model.hctr || 0, 10);
};

Extract.prototype.extractModel = function(el) {
	var info = this.json[el.getAttribute('id')];
	// Did I make this post?
	if (this.mine[info.num])
		info.mine = true;
	return info;
};
