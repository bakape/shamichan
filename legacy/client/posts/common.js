module.exports = imager.Hidamari.extend({
	renderTime() {
		this.el.query('time').outerHTML = oneeSama.time(this.model.get('time'));
	},

	// Self-delusion tripfag filter
	anonymise() {
		this.el.query('.name').innerHTML = `<b class="name">${lang.anon}<b>`;
	},

	// Restore regular name
	renderName() {
		this.el.query('.name').outerHTML = oneeSama.name(this.model.attributes);
	},

	renderModerationInfo(info) {
		const el = this.getContainer();
		el.query('.modLog').remove();
		el.query('blockquote').before(util.parseDOM(oneeSama.modInfo(info)));
	},

	getContainer() {
		return this.el.query('.container');
	},

	renderBan() {
		const el = this.getContainer();
		el.query('.banMessage').remove();
		el.query('blockquote').after(util.parseDOM(oneeSama.banned()));
	},
});
