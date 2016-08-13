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
	// Calling a method is always less overhead than binding a dedicated
	// listener for each post's image
	setImage(image, silent) {
		this.set('image', image);
		if (!silent)
			this.dispatch('renderImage', image);
	},
	setSpoiler(spoiler, info) {
		let image = this.get('image');
		image.spoiler = spoiler;
		this.dispatch('renderImage', image);
		this.moderationInfo(info);
	},
	removeImage(info) {
		// Staff won't have the image removed, but rerendered with
		// indication, that it has been deleted and extra information
		this.moderationInfo(info)
			|| this.unset('image').dispatch('renderImage');
	},
	deletePost(info) {
		this.moderationInfo(info) || this.remove();
	},
	setBan(display, info) {
		// Displaying the 'USER WAS BANNED FOR THIS POST' message and
		// renderring the moderation info are independant actions
		if (display)
			this.set('ban', true).dispatch('renderBan');
		this.moderationInfo(info);
	},
	// Add info about the moderation action taken. This is only used on
	// authenticated staff clients, but for sanity, lets keep it here in
	// common model methods.
	moderationInfo(info) {
		if (!info)
			return false;
		const mod = this.get('mod') || [];
		mod.push(info);
		this.set('mod', mod)
			.dispatch('renderModerationInfo', mod);
		return true;
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
	toggleLocked(val, info) {
		this.moderationInfo(info);
		this.set('locked', val).dispatch('renderLocked', val);
	}
});
