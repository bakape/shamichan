/*
 * OP and thread related logic
 */

var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	main = require('../main'),
	oneeSama = main.oneeSama,
	postCommon = require('./common'),
	state = require('../state');

var Section = module.exports = Backbone.View.extend({
	tagName: 'section',

	initialize: function () {
		// On the live page only
		if (this.$el.is(':empty'))
			this.render();
		else
			this.renderOmit(null, this.model.get('omit'));

		this.listenTo(this.model, {
			'change:locked': this.renderLocked,
			remove: this.remove,
			shiftReplies: this.shiftReplies,
			'change:omit': this.renderOmit,
			bump: this.bumpThread
		});
		this.listenToOnce(this.model, {
			'add': this.renderRelativeTime
		});
		this.initCommon();
	},

	render: function() {
		let attrs = this.model.attributes;
		oneeSama.links = attrs.links;
		this.setElement(oneeSama.monomono(attrs).join(''));
		this.insertToTop();
		// Insert reply box into the new thread
		let $reply = $(oneeSama.replyBox());
		if (state.ownPosts.hasOwnProperty(attrs.num)
			|| !!main.request('postForm')
		)
			$reply.hide();
		this.$el.after($reply, '<hr>');
		return this;
	},

	insertToTop: function() {
		this.$el.insertAfter(main.$threads.children('aside').first());
	},

	renderHide: function (model, hide) {
		this.$el.next('hr').andBack().toggle(!hide);
	},

	renderLocked: function (model, locked) {
		this.$el.toggleClass('locked', !!locked);
	},

	remove: function () {
		this.$el.next('hr').addBack().remove();
		this.stopListening();
		return this;
	},

	/*
	 Remove the top reply on board pages, if over limit, when a new reply is
	 added
	 */
	shiftReplies: function(postForm) {
		if (state.page.get('thread'))
			return;
		var replies = this.model.get('replies'),
			lim = state.hotConfig.get('ABBREVIATED_REPLIES');
		if (postForm)
			lim--;
		let image_omit = this.model.attributes.image_omit;
		for (let i = replies.length; i > lim; i--) {
			let post = state.posts.get(replies.shift());
			if (!post)
				continue;
			/*
			 Nothing tracks changes on image_omit, but we want omit changes to
			 properly trigger change events.
			  */
			if (post.get('image'))
				image_omit++;
			this.model.set('omit', this.model.get('omit') + 1 );
			post.remove();
		}
	},

	// Posts and images omited indicator
	renderOmit: function(model, omit) {
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
	bumpThread: function() {
		this.$el.detach();
		this.insertToTop();
	}
});

// Extend with common mixins
_.extend(Section.prototype, postCommon);
