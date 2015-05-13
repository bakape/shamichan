/*
 * Hover previews
 */

var $ = require('jquery'),
	Backbone = require('backbone'),
	main = require('./main'),
	options = require('./options'),
	article = require('./posts').Article,
	state = require('./state');

// Centralised mousemove target tracking
var mousemove = exports.mousemove = new Backbone.Model({
	id: 'mousemove',
	/*Logging only the target isn't a option because change:target doesn't seem
	to fire in some cases where the target is too similar for example changing
	between two post links (>>XXX) directly*/
	event: null
});

var ImageHoverView = Backbone.View.extend({
	initialize: function() {
		this.listenTo(this.model,'change:event', this.check);
		this.listenTo(options, 'imageClicked', function() {
			this.$el.empty();
		});
	},

	check: function(model, event) {
		// Disabled in options
		if (!options.get('imageHover'))
			return;
		var $target = $(event.target);
		if (!$target.is('img') || $target.hasClass('expanded'))
			return this.$el.children().remove("img,video");
		const src = $target.closest('a').attr('href'),
			isWebm = /\.webm$/.test(src);
		// Nothing to preview for PDF or MP3
		if (/\.pdf$/.test(src)
			|| /\.mp3$/.test(src)
			|| (isWebm && !options.get('webmHover'))
		)
			return this.$el.empty();
		$(isWebm ? '<video/>' : '<img/>', {
			src: src,
			autoplay: true,
			loop: true
		}).appendTo(this.$el.empty());
	}
});
var PostPreview = article.extend({
	initialize: function () {
		this.listenTo(this.model, {
			'change:body': this.update,
			'change:image': this.update,
			'change:editing': this.renderEditing
		});
		this.render().$el.addClass('preview');
		this.initCommon().update();
	},

	update: function() {
		postHover.render(this.$el);
	}
});
var HoverPostView = Backbone.View.extend({
	previewView: null,
	targetPos:null,
	initialize: function() {
		this.listenTo(this.model,'change:event', this.check);
	},
	check: function(model, event) {
		let $target = $(event.target);
		if (!$target.is('a.history'))
			return;
		const m = event.target.text.match(/^>>(\d+)/);
		if (!m)
			return;
		let post = state.posts.get(m[1]);
		if (!post)
			return;
		this.targetPos = $target.position();
		this.previewView = new PostPreview({model: post});
		$target.one('mouseleave click', () => this.remove());
	},
	remove: function() {
		if (this.previewView) {
			this.targetPos = null;
			this.previewView.remove();
			this.stopListening(this.previewView);
			this.$el.children().remove('.preview');
			this.previewView = null;
		}
	},
	render: function($el) {
		$el.css(this.position($el));
		$el.appendTo(this.$el.empty());
	},

	position: function($el) {
		$el.hide();
		$(document.body).append($el);
		var w = $el.width();
		var h = $el.height();
		$el.detach().show();

		var $w = $(window);
		var l = this.targetPos.left -$w.scrollLeft();
		var t = this.targetPos.top -$w.scrollTop()-h-17;

		//If it get cut at the top, put it below the link
		if(t<0)
			t+=h+34;

		//if it gets cut to the right push it to the left.
		var overflowR = l+w-$w.innerWidth();
		if(overflowR>-30)
			l = Math.max(0,l-overflowR-30);

		return {left: l,top: t};
	}
});

if (!main.isMobile) {
	var ltarget;
	main.$doc.on('mousemove', function(e) {
		if(e.target!=ltarget) {
			mousemove.set('event', e);
			ltarget= e.target;
		}
	});
	var $hover =document.getElementById('hover_overlay');
	new ImageHoverView({
		model: mousemove,
		el: $hover
	});
	var postHover = new HoverPostView({
		model: mousemove,
		el: $hover
	});
}
