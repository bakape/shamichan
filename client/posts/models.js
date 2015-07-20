/*
General post backbone models
 */

let main = require('../main'),
	{_, Backbone, state} = main;

exports.Post = Backbone.Model.extend({
	idAttribute: 'num',
	initialize() {
		state.posts.add(this);
	},
	// Proxy commands to the view(s). Using a central channel helps us reduce
	// listener count overhead.
	dispatch(command, ...args) {
		this.trigger('dispatch', command, ...args);
	},
	remove() {
		this.stopListening();
		// Remove view
		this.dispatch('remove');
		// Remove from post collection
		state.posts.remove(this);
	},
	update(frag, extra) {
		let updates = {
			body: this.get('body') + frag,
			state: extra.state
		};
		const {links, dice} = extra;
		if (links)
			// No listeners, so can be silent. We don't even use it at the
			// moment, but let's keep it arround for model consistency for now.
			_.extend(this.get('links'), links);
		if (dice)
			updates.dice = (this.get('dice') || []).concat(dice);
		this.set(updates);
	}
	,
	// Calling a method is always less overhead than binding a dedicated
	// listener for each post's image
	setImage(image, silent) {
		this.set('image', image);
		if (!silent)
			this.dispatch('renderImage', image);
	},
	setSpoiler(spoiler) {
		let image = this.get('image');
		image.spoiler = spoiler;
		this.dispatch('renderImage', image);
	},
	removeImage() {
		// Moderators won't have the image removed, but rerendered with
		// indication, that it has been deleted.
		if (main.ident)
			this.get('image').imgDeleted = true;
		else
			this.unset('image');
		this.dispatch('renderImage');
	},
	addBacklink(num, op) {
		let backlinks = this.get('backlinks') || {};
		backlinks[num] = op;
		this.set({backlinks})
			.dispatch('renderBacklinks', backlinks);
	}
});

exports.Thread = exports.Post.extend({
	defaults: {
		replies: [],
		omit: 0,
		image_omit: 0
	},
	initialize() {
		// Omitted images can only be calculated, if there are omitted posts
		if (this.get('omit'))
			this.getImageOmit();
		state.posts.add(this);
	},
	remove() {
		this.stopListening();
		this.dispatch('remove');
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
	getImageOmit() {
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
	},
	toggleLocked(val) {
		this.dispatch('renderLocked', val);
	}
});
