/*
General post backbone models
 */
'use strict';

var _ = require('underscore'),
	Backbone = require('backbone'),
	main = require('../main'),
	state = main.state;

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
	addLinks: function(links){
		if(!links)
			return;
		var old = this.get('links');
		if(!old)
			return this.set({links: links});
		_.extend(old,links);
		this.set({links:old});
		// If we get here we changed something for sure, but as we are using the
		// same ref backbone will ignore it so we have to force the event to
		// trigger.
		this.trigger('change:links', this, old);
	},
	// Pass this post's links to the central model
	forwardLinks: function(model, links) {
		var old, newLinks;
		const num = this.get('num'),
			op = this.get('op') || num;
		const mine = state.mine.read_all();
		for (let key in links) {
			if (mine[key])
				main.command('repliedToMe', this);
			old = state.linkerCore.get(key);
			newLinks = old ? _.clone(old) : {};
			newLinks[num] = op;
			state.linkerCore.set(key, newLinks);
		}
	}
});

exports.Thread = exports.Post.extend({
	defaults: {
		replies: [],
		omit: 0,
		image_omit: 0
	},
	initialize: function() {
		// Omitted images can only be calculated, if there are omitted posts
		if (this.get('omit'))
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
