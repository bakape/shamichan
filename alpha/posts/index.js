/*
 * Contains the core models and views for threads and posts
 */

var Backbone = require('backbone'),
	common = require('../../common'),
	main = require('../main');

exports.Article = require('./article');
exports.Section = require('./section');

exports.PostModel = Backbone.Model.extend({
	initialize: function() {
		main.posts.add(this);
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
		main.posts.add(this);
		main.threads.add(this);
	},
	idAttribute: 'num',
	replies: new main.PostCollection()
});
