/*
 * Contains the core models and views for threads and posts
 */

var Backbone = require('backbone'),
	common = require('../../common'),
	main = require('../main');

exports.Article = require('./article');
exports.Section = require('./section');

var PostCollection = Backbone.Collection.extend({
	idAttribute: 'num'
});

// All posts currently displayed
var posts = exports.posts = new PostCollection(),
	/*
	 * All threads currently displayed. Threads are also posts, so they are in
	 * both collections. This seperation is needed, not to search through all
	 * posts, to find a thread.
	 */
	threads = exports.threads = new PostCollection();

exports.PostModel = Backbone.Model.extend({
	initialize: function() {
		posts.add(this);
	},
	idAttribute: 'num'
});

exports.ThreadModel = Backbone.Model.extend({
	initialize: function(args) {
		if (args.replies)
			this.replies.add(args.replies);
		// Propagate model destruction to reply collection
		this.listenTo(this, {
			destroy: function() {
				this.replies.model.destroy();
			}
		});
		posts.add(this);
		threads.add(this);
	},
	idAttribute: 'num',
	replies: new PostCollection()
});
