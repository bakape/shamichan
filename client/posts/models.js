/*
General post backbone models
 */

var Backbone = require('backbone'),
	state = require('../state');

exports.Post = Backbone.Model.extend({
	idAttribute: 'num',

	initialize: function() {
		state.posts.add(this);
	},

	remove: function() {
		this.stopListening();
		// Remove view
		this.trigger('remove');
		// Remove from post collection
		state.posts.remove(this);
	}
});

exports.Thread = exports.Post.extend({
	initialize: function() {
		if (!this.get('omit')) {
			this.set({
				omit: 0,
				image_omit: 0
			});
		}
		else
			this.getImageOmit();
		state.posts.add(this);
	},

	remove: function() {
		this.stopListening();
		this.trigger('remove');
		state.posts.remove(this);

		// Propagate model removal to all replies
		this.get('replies').forEach(function(num) {
			var model = state.posts.get(num);
			if (model)
				model.remove();
		});
	},

	/*
	 With the current renderring and storage implementations we can not get the
	 image omit count during the server-side render.
	 */
	getImageOmit: function() {
		var model,
			image_omit = this.get('imgctr') -1;
		for (var num of this.get('replies')) {
			model = state.posts.get(num);
			if (!model)
				continue;
			if (model.get('image'))
				image_omit--;
		}
		this.set('image_omit', image_omit);
	}
});
