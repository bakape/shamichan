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
		$(isWebm ? '<video/>' : '<images/>', {
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
});

let ltarget, postHover;
if (!main.isMobile) {
	main.defer(function() {
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
