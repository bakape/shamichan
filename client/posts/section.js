/*
 * OP and thread related logic
 */

let main = require('../main'),
	postCommon = require('./common'),
	{$, _, Backbone, oneeSama, state} = main;

var Section = module.exports = Backbone.View.extend({
	tagName: 'section',
	render() {
		let attrs = this.model.attributes;
		this.setElement(oneeSama.section(attrs).join('')).insertIntoDOM();
		// Insert reply box into the new thread
		let $reply = $(oneeSama.replyBox());
		if (state.ownPosts.hasOwnProperty(attrs.num)
			|| !!main.request('postForm')
		)
			$reply.hide();
		this.$el.append($reply)
			.next('hr').remove();
		return this;
	},
	insertIntoDOM() {
		this.$el.insertAfter(main.$threads.children('aside').first())
			.after('<hr>');
		this.fun();
	},
	renderLocked(locked) {
		this.$el.toggleClass('locked', locked);
	},
	remove() {
		this.$el.next('hr').addBack().remove();
		this.stopListening();
		return this;
	},
	/*
	 Remove the top reply on board pages, if over limit, when a new reply is
	 added
	 */
	shiftReplies(postForm) {
		let attrs = this.model.attributes,
			lim = state.hotConfig.get('ABBREVIATED_REPLIES'),
			replies = attrs.replies,
			changed;
		if (postForm)
			lim--;
		// Need a static length, because the original array get's modified
		const len = replies.slice().length;
		for (let i = len; i > lim; i--) {
			let post = state.posts.get(replies.shift());
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
	renderOmit(omit, image_omit) {
		if (typeof omit === 'undefined') {
			const attrs = this.model.attributes;
			omit = attrs.omit;
			image_omit = attrs.image_omit;
		}
		if (omit === 0)
			return;
		if (!this.$omit) {
			this.$omit = $('<span class="omit"/>')
				.insertAfter(this.$el.children('blockquote'));
		}
		const page = state.page.attributes;
		var html = oneeSama.lang.abbrev_msg(omit,
			this.model.get('image_omit'),
			// [See All] link URL
			page.thread && page.href.split('?')[0]
		);
		this.$omit.html(html);
	},
	// Move thread to the top of the page
	bumpThread() {
		this.$el.next('hr').remove();
		this.$el.detach();
		this.insertIntoDOM();
	},
	// TEMP: Stub until we unify the DOM structure of OPs and replies
	renderEditing() {

	}
});

// Extend with common mixins
_.extend(Section.prototype, postCommon);
