module.exports = PostCommon.extend({
	renderLocked(locked) {
		this.el.classList[locked ? 'add' : 'remove']('locked');
	},
	// Posts and images omited indicator
	renderOmit(omit = this.model.get('omit'),
		image_omit = this.model.get('image_omit')
	) {
		if (omit === 0)
			return;
		const {thread, href} = state.page.attributes;
		this.el.query('.omit').innerHTML = oneeSama.lang.abbrev_msg(omit,
			this.model.get('image_omit'), thread && href.split('?')[0]);
	},
});
