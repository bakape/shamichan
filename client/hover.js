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
			'change:body': this.render,
			'change:editing': this.render,
			'change:image': this.render
		});
		this.initCommon();
	},
	render: function () {
		main.oneeSama.links = this.model.get('links');
		this.setElement(main.oneeSama.mono(this.model.attributes));
		this.$el.addClass('preview').append(this.$backlinks);
		this.trigger("update",this.$el);
		return this;
	}
});
var HoverPostView = Backbone.View.extend({
	previewView: null,
	targetPos:null,
	initialize: function() {
		this.listenTo(this.model,'change:event', this.check);
	},
	check: function(model, event) {
		var $target = $(event.target);
		if ($target.is('a.history')){
			var m = event.target.text.match(/^>>(\d+)/);
			if (m) {
				let post = state.posts.get(m[1]);
				if (!post)
					return;
				this.previewView = new PostPreview({model: post});
				this.targetPos = $target.position();

				//If the post changes we update the preview
				this.listenTo(this.previewView, 'update', this.render);
				this.previewView.render();

				var modelref = this;
				$target.one('mouseleave click',function(){
					modelref.remove();
				});
			}
		}
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
	new HoverPostView({
		model: mousemove,
		el: $hover
	});
}
