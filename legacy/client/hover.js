/*
 * Hover previews
 */

let main = require('./main'),
	{Article} = main.posts,
	{$, Backbone, etc, options, state} = main;

// Centralised mousemove target tracking
/*Logging only the target isn't a option because change:target doesn't seem
 to fire in some cases where the target is too similar for example changing
 between two post links (>>XXX) directly
 */
const mousemove = new Backbone.Model();

let ImageHoverView = Backbone.View.extend({
	initialize() {
		this.listenTo(this.model,'change:event', this.check);
		main.reply('imager:clicked', () => this.$el.empty());
	},
	check(model, event) {
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

let PostPreview = Article.extend({
	initialize(args) {
		this.parentNum = args.parentNum;
		this.listenTo(this.model, 'dispatch', this.redirect)
			.render().$el.addClass('preview');
		this.clientInit()
		this.update();
	},
	update() {
		if (!this.num)
			this.num = etc.getID(this.el);
		postHover.render(this.$el, this.parentNum);
	}
});

let HoverPostView = Backbone.View.extend({
	initialize() {
		this.listenTo(this.model,'change:event', this.check);
	},
	check(model, event) {
		let $target = $(event.target);
		if (!$target.is('a.history'))
			return;
		const m = $target.text().match(/^>>(\d+)/);
		if (!m)
			return;
		let post = state.posts.get(m[1]);
		if (!post)
			return;
		this.targetPos = $target.offset();
		this.previewView = new PostPreview({
			model: post,
			parentNum: etc.getID(event.target)
		});
		$target.one('mouseleave click', () => this.remove());
	},
	remove() {
		if (!this.previewView)
			return;
		this.targetPos = null;
		this.previewView.remove();
		this.stopListening(this.previewView);
		this.$el.children().remove('.preview');
		this.previewView = null;
	},
	render($el, num) {
		// Striped underline links from the parent post
		$el.find('a.history')
			.filter(function () {
				return this.text.includes('>>' + num);
			})
			.each(function () {
				$(this).addClass('referenced');
			});
		$el.css(this.position($el)).appendTo(this.$el.empty());
	},
	position($el) {
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

let ltarget, postHover;
if (!main.isMobile) {
	main.defer(function() {
		main.$doc.on('mousemove', function(e) {
			if(e.target!=ltarget) {
				mousemove.set('event', e);
				ltarget= e.target;
			}
		});
		let hover = document.getElementById('hover_overlay');
		new ImageHoverView({
			model: mousemove,
			el: hover
		});
		postHover = new HoverPostView({
			model: mousemove,
			el: hover
		});
	});
}
