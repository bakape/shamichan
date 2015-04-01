/*
 * Non-OP posts
 */

var $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	imager = require('./imager'),
	main = require('../main'),
	options = require('../options'),
	postCommon = require('./common');

var Article = module.exports = Backbone.View.extend({
	tagName: 'article',

	initialize: function () {
		this.listenTo(this.model, {
			'change:backlinks': this.renderBacklinks,
			'change:editing': this.renderEditing,
			'change:image': this.renderImage,
			'removeSelf': this.bumplessRemove
		});
		this.initCommon();
		if (options.get('postUnloading') && state.page.get('thread')) {
			this.listenTo(this.model, {
				'add': unloadTopPost
			});
		}
	},

	render: function () {
		var html = main.oneeSama.mono(this.model.attributes);
		this.setElement($($.parseHTML(html)).filter('article')[0]);
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
			this.$('blockquote')[0].normalize();
	},

	renderHide: function (model, hide) {
		this.$el.toggle(!hide);
	},

	renderImage: function (model, image) {
		var hd = this.$('header'), fig = this.$('figure');
		if (!image)
			fig.remove();
		else if (hd.length && !fig.length) {
			/* Is this focus business necessary here? */
			var focus = get_focus();

			insert_image(image, hd, false);

			if (focus)
				focus.focus();
			this.autoExpandImage();
		}
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