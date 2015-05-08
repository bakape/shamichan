/*
General post backbone models
 */
'use strict';

var _ = require('underscore'),
	Backbone = require('backbone'),
	state = require('../state');

exports.Post = Backbone.Model.extend({
	idAttribute: 'num',

	initialize: function() {
		this.initCommon();
	},

	// Initialisation logic common to both replies and threads
	initCommon: function() {
		state.posts.add(this);
		const links = this.get('links');
		if (links)
			this.forwardLinks(null, links);
		this.listenTo(this, 'change:links', this.forwardLinks);
	},

	remove: function() {
		this.stopListening();
		// Remove view
		this.trigger('remove');
		// Remove from post collection
		state.posts.remove(this);
	},

	// Pass this post's links to the central model
	forwardLinks: function(model, links) {
		var old, newLinks;
		const num = this.get('num'),
			op = this.get('op') || num;
		for (let key in links) {
			old = state.linkerCore.get(key);
			newLinks = old ? _.clone(old) : {};
			newLinks[num] = op;
			state.linkerCore.set(key, newLinks);
		}
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
		this.initCommon();
	},

	remove: function() {
		this.stopListening();
		this.trigger('remove');
		state.posts.remove(this);

		// Propagate model removal to all replies
		const replies = this.get('replies');
		for (let i = 0, lim = replies.length; i < lim; i++) {
			let model = state.posts.get(replies[i]);
			if (model)
				model.remove();
		}
	},

	/*
	 With the current renderring and storage implementations we can not get the
	 image omit count during the server-side render.
	 */
	getImageOmit: function() {
		let image_omit = this.get('imgctr') -1;
		const replies = this.get('replies');

		for (let i = 0, lim = replies.length; i < lim; i++) {
			let model = state.posts.get(replies[i]);
			if (!model)
				continue;
			if (model.get('image'))
				image_omit--;
		}
		this.set('image_omit', image_omit);
	}
});
