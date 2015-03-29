/*
 * Core Backbone models
 */
var _ = require('underscore'),
	Backbone = require('backbone'),
	imager = require('./imager'),
	main = require('./main'),
	options = require('./options'),
	time = require('./time');

var PostCollection = Backbone.Collection.extend({
	idAttribute: 'num'
});

// All posts currently displayed
var posts = exports.posts = new PostCollection();

var Section = Backbone.View.extend({
	tagName: 'section',

	initialize: function () {
		this.listenTo(this.model, {
			'change:locked': this.renderLocked,
			destroy: this.remove,
		});
		this.listenToOnce(this.model, {
			'add': this.renderRelativeTime
		});
		this.listenTo(this.model.get('replies'), {
			remove: this.removePost,
		});
		this.initCommon();
	},

	replies: new PostCollection(),

	renderHide: function (model, hide) {
		this.$el.next('hr.sectionHr').andSelf().toggle(!hide);
	},

	renderLocked: function (model, locked) {
		this.$el.toggleClass('locked', !!locked);
	},

	remove: function () {
		var replies = this.model.get('replies');
		replies.each(function (post) {
			clear_post_links(post, replies);
		});
		replies.reset();

		this.$el.next('hr.sectionHr').andSelf().remove();
		// Remove from all Posts collection
		Posts.remove(this.model);
		this.stopListening();
	},

	removePost: function (model) {
		model.trigger('removeSelf');
	},
});

// XXX: Move into own views module once more substantial
var Article = Backbone.View.extend({
	tagName: 'article',
	initialize: function () {
		this.listenTo(this.model, {
			'change:backlinks': this.renderBacklinks,
			'change:editing': this.renderEditing,
			'change:image': this.renderImage,
			'removeSelf': this.bumplessRemove
		});
		this.initCommon();
		if (options.get('postUnloading') && CurThread) {
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

// Common to both Articles and sections
var PostMixins = {
	initCommon: function(){
		this.listenTo(this.model, {
			'change:hide': this.renderHide,
			'spoiler': this.renderSpoiler
		});
		// Models get added to multiple collections. Prevent duplication by
		// calling only once
		this.listenToOnce(this.model, {
			add: function() {
				this.renderRelativeTime();
				this.fun();
				// Anonymise on post insertion
				if (options.get('anonymise'))
					this.toggleAnonymisation(null, true);
			}
		});
		this.listenTo(options, {
			'change:thumbs': this.changeThumbnailStyle,
			'change:noSpoilers': this.toggleSpoiler,
			'change:autogif': this.toggleAutogif,
			'change:anonymise': this.toggleAnonymisation
		});
		this.listenTo(imager.massExpander, {
			'change:expand': this.toggleImageExpansion,
		});
	},

	renderRelativeTime: function(){
		if (main.oneeSama.rTime){
			var $time = this.$el.find('time').first();
			const t = time.date_from_time_el($time[0]).getTime();
			var timer = setInterval(function(){
				$time.html(main.oneeSama.relative_time(t, new Date().getTime()));
			}, 60000);
			this.listenToOnce(this.model, 'removeSelf', function(){
				clearInterval(timer);
			});
		}
	},

	fun: function(){
		// Fun goes here
	},

	// Self-delusion tripfag filter
	toggleAnonymisation: function(model, toggle) {
		var $el = this.$el.find('>header>b'),
			name = this.model.get('name');
		if (toggle)
			$el.text(common.ANON);
		// No need to change, if no name
		else if (name)
			$el.text(name);
	}
};

// Extend with common mixins
_.extend(Section.prototype, imager.Hidamari, PostMixins);
_.extend(Article.prototype, imager.Hidamari, PostMixins);

// Centralised mouseover target tracking
var mouseover = exports.mouseover = new Backbone.Model({target: null});

if (!isMobile) {
	$DOC.on('mouseover', function(e) {
		mouseover.set('target', e.target);
	});
}