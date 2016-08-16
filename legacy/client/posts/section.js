module.exports = PostCommon.extend({
	renderLocked(locked) {
		this.el.classList[locked ? 'add' : 'remove']('locked');
	},
});
