/*
 * OP and thread related logic
 */

const main = require('../main'),
	PostCommon = require('./common'),
	{$, _, Backbone, util, oneeSama, state} = main;

module.exports = PostCommon.extend({
	tagName: 'section',
	render() {
		const attrs = this.model.attributes;
		this.setElement(oneeSama.section(attrs)).insertIntoDOM();

		// Insert reply box into the new thread
		const reply = util.parseDOM(oneeSama.replyBox());
		if (attrs.num in state.ownPosts || !!main.request('postForm'))
			reply.style.display = 'none';
		this.el.append(reply);

		// Remove next <hr>
		this.el.nextElementSibling.remove();
		return this;
	},
	insertIntoDOM() {
		main.$threads[0].query('aside')
			.after(this.el, document.createElement('hr'));
		this.fun();
	},
	renderLocked(locked) {
		this.el.classList[locked ? 'add' : 'remove']('locked');
	},
	remove() {
		// Remove next <hr>
		this.el.nextElementSibling.remove();
		this.el.remove();
		this.stopListening();
		return this;
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
