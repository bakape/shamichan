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
	/*
	 Remove the top reply on board pages, if over limit, when a new reply is
	 added
	 */
	shiftReplies(postForm) {
		const attrs = this.model.attributes,
			{replies} = attrs;
		let lim = state.hotConfig.get('ABBREVIATED_REPLIES'),
			changed;
		if (postForm)
			lim--;
		// Need a static length, because the original array get's modified
		const len = replies.slice().length;
		for (let i = len; i > lim; i--) {
			const post = state.posts.get(replies.shift());
			if (!post)
				continue;
			if (post.get('image'))
				attrs.image_omit++;
			attrs.omit++;
			changed = true;
			post.remove();
		}
		if (changed)
			this.renderOmit(attrs.omit, attrs.image_omit)
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
	// Move thread to the top of the page
	bumpThread() {
		this.el.nextElementSibling.remove();
		this.el.remove();
		this.insertIntoDOM();
	}
});
