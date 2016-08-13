/*
 * Common methods to both OP and regular posts
 */

const main = require('../main'),
	imager = require('./imager'),
	{_, Backbone, common, util, lang, oneeSama, options, state} = main;

module.exports = imager.Hidamari.extend({
	className: 'glass',
	// One-way communication channel to the model
	initialize() {
		this.listenTo(this.model, 'dispatch', this.redirect);
	},
	renderTime() {
		this.el.query('time').outerHTML = oneeSama.time(this.model.get('time'));
	},
	// Admin JS injections
	fun() {
		// Fun goes here
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
