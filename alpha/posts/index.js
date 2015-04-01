/*
 * Contains the core models and views for threads and posts
 */

var Backbone = require('backbone');

exports.Article = require('./article');
exports.Section = require('./section');

var PostModel = exports.PostModel = Backbone.Model.extend({
	idAttribute: 'num'
});

var PostCollection = Backbone.Collection.extend({
	idAttribute: 'num'
});

var ThreadModel = exports.ThreadModel = PostModel.extend({
	replies: new PostCollection()
});

// All posts currently displayed
var posts = exports.posts = new PostCollection(),
	/*
	 * All threads currently displayed. Threads are also posts, so they are in
	 * both collections. This seperation is needed, not to search through all
	 * posts, to find a thread.
	 */
	threads = exports.threads = new PostCollection();