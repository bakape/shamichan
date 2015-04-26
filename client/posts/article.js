/*
 * Non-OP posts
 */

var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	hover = require('../hover'),
	imager = require('./imager'),
	main = require('../main'),
	options = require('../options'),
	postCommon = require('./common'),
	state = require('../state');

var Article = module.exports = Backbone.View.extend({
	tagName: 'article',

	initialize: function () {
		/*
		 * XXX: A bit ineficient, because first an empty element is renderred
		 * and then a proper one.
		 *
		 * An element is not empty only on postForms
		 */
		if (this.$el.is(':empty'))
			this.render();
		this.listenTo(this.model, {
			'change:backlinks': this.renderBacklinks,
			'change:editing': this.renderEditing,
			'change:image': this.renderImage,
			removeSelf: this.bumplessRemove,
			destroy: this.remove
		});
		this.initCommon();
		/* TEMP: Disabled for now
		if (options.get('postUnloading') && state.page.get('thread')) {
			this.listenTo(this.model, {
				'add': unloadTopPost
			});
		}
		*/
	},

	render: function () {
		/*
		 * Pass this model's links to oneeSama for renderring. The reason we
		 * don't use the links attribute directly in OneeSama is different
		 * rendering pathways on the server and client.
		 * XXX: Unify this shit.
		 */
		main.oneeSama.links = this.model.get('links');
		this.setElement(main.oneeSama.mono(this.model.attributes));
		// Insert into section
		$('#' + this.model.get('op'))
			.children('blockquote,.omit,form,article[id]:last')
			.last()
			.after(this.$el);
		return this;
	},

	renderBacklinks: function () {
		if (options.get('backlinks'))
			return this; /* ought to disconnect handler? */
		var backlinks = this.model.get('backlinks');
		var $list = this.$el.find('small');
		if (!backlinks || !backlinks.length) {
			$list.remove();
			return this;
		}
		if (!$list.length)
			$list = $('<small/>', {text: 'Replies:'}).appendTo(
					this.$el);
		// TODO: Sync up DOM gracefully instead of clobbering
		$list.find('a').remove();
		backlinks.forEach(function (num) {
			var $a = $('<a/>', {href: '#'+num, text: '>>'+num});
			$list.append(' ', $a);
		});
		return this;
	},

	renderEditing: function (model, editing) {
		this.$el.toggleClass('editing', !!editing);
		if (!editing)
			this.$el.children('blockquote')[0].normalize();
	},

	renderHide: function (model, hide) {
		this.$el.toggle(!hide);
	},

	// To not shift the scroll position on remove
	// TODO: Rework, once scrolling code is done
	bumplessRemove: function(){
		const pos = $(window).scrollTop();
		if (!at_bottom() && this.$el.offset().top < pos)
			// Not sure why we need the extra 2 pixels, but we do
			$(window).scrollTop(pos - this.$el.outerHeight() - 2);
		Posts.remove(this.model);
		this.remove();
	},
});

// Extend with common mixins
_.extend(Article.prototype, imager.Hidamari, postCommon);

// Keeps threads non-laggy by keeping displayed post count within lastN
function unloadTopPost(){
	var m = location.search.match(/last=(\d+)/);
	const threadNum = state.page.get('thread');
	if (!m
		|| $(hover.mouseover.get('target')).is('a, img, video')
		|| threadNum > 0
	) {
		return;
	}
	var	thread = state.getThread(threadNum);
	if (thread.replies.length <= parseInt(m[1], 10) + 5)
		return;
	state.posts.get(thread.replies.shift()).destroy();
	var $omit = $('.omit');
	if (!$omit.length){
		$omit = $('\t<span/>', {'class': 'omit'}).text(main.lang.abbrev_msg(1));
		$omit.append(common.action_link_html(threadNum
			+ location.hash, 'See all')+'\n');
		$('section>blockquote').after($omit);
	}
	else {
		var m = $omit.html().match(/^(\d+)(.*)/);
		$('.omit').html(parseInt(m[1])+1+m[2]);
	}
	unloadTopPost();
}
