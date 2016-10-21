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
