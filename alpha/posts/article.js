/*
 * Non-OP posts
 */

var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	imager = require('./imager'),
	main = require('../main'),
	options = require('../options'),
	postCommon = require('./common'),
	state = require('../state');

var Article = module.exports = Backbone.View.extend({
	tagName: 'article',

	initialize: function () {
		if (!this.el)
			this.render();
		this.listenTo(this.model, {
			'change:backlinks': this.renderBacklinks,
			'change:editing': this.renderEditing,
			'change:image': this.renderImage,
			removeSelf: this.bumplessRemove,
			destroy: this.remove
		});
		this.initCommon();
		if (options.get('postUnloading') && state.page.get('thread')) {
			this.listenTo(this.model, {
				'add': unloadTopPost
			});
		}
	},

	render: function () {
		this.setElement($($.parseHTML(main.oneeSama.mono(this.model.attributes)))
			.filter('article')[0]);
		// Insert into section
		$('#' + this.model.get('op'))
			.children('blockquote,.omit,form,article[id]:last')
			.last()
			.after(this.$el);
		return this;
	},

	renderBacklinks: function () {
		if (options.get('nobacklinks'))
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