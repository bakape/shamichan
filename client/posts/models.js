/*
General post backbone models
 */

var Backbone = require('backbone'),
	state = require('../state');

exports.Post = Backbone.Model.extend({
	initialize: function() {
		state.posts.add(this);
	},

	destroy: function() {
		this.stopListening();
		// Remove from post collection
		state.posts.trigger('destroy', this);
	},

	idAttribute: 'num'
});

exports.Thread = Backbone.Model.extend({
	initialize: function(args) {
		if (args.replies) {
			this.replies = args.replies;
			// Lighten up the model
			this.unset('replies', {silent: true});
		}

		state.posts.add(this);
		state.threads.add(this.get('num'));
	},

	destroy: function() {
		this.stopListening();
		state.posts.trigger('destroy', this);

		// Propagate model destruction to reply collection
		this.replies.forEach(function(num) {
			var model = state.posts.get(num);
			if (model)
				model.destroy();
		});
	},

	idAttribute: 'num',
	replies: []
});
