exports.Post = Backbone.Model.extend({
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
	toggleLocked(val, info) {
		this.moderationInfo(info);
		this.set('locked', val).dispatch('renderLocked', val);
	}
});
