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
