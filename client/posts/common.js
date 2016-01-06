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
	// Extra initialisation logic for posts renderred client-side
	clientInit() {
		if (options.get('anonymise'))
			this.anonymise();
		return this;
	},
	// Proxy to the appropriate method
	redirect(command, ...args) {
		this[command](...args);
	},
	// Update the post's text body
	updateBody(frag) {
		if (!this.blockquote)
			this.blockquote = this.el.query('blockquote');

		// This will rerender the HTML content on each update. Might be
		// some overhead involved, but simplifies live updates greatly.
		const model = this.model.attributes;
		this.blockquote.innerHTML = oneeSama.setModel(model).body(model.body);
	},
	renderTime() {
		this.el.query('time').outerHTML = oneeSama.time(this.model.get('time'));
	},
	renderBacklinks(links) {
		this.el.query('small').innerHTML = oneeSama.backlinks(links);
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
	renderEditing(editing) {
		const {el} = this;
		if (editing)
			el.classList.add('editing');
		else {
			el.classList.remove('editing');
			el.query('blockquote').normalize();
		}
	}
});
