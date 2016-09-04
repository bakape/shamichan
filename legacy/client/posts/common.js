module.exports = imager.Hidamari.extend({
	renderModerationInfo(info) {
		const el = this.getContainer();
		el.query('.modLog').remove();
		el.query('blockquote').before(util.parseDOM(oneeSama.modInfo(info)));
	},

	renderBan() {
		const el = this.getContainer();
		el.query('.banMessage').remove();
		el.query('blockquote').after(util.parseDOM(oneeSama.banned()));
	},
});
