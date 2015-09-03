/*
 * Reply posts
 */

let main = require('../main'),
	postCommon = require('./common'),
	{$, _, Backbone, options, state} = main;

var Article = module.exports = Backbone.View.extend({
	tagName: 'article',
	render() {
		this.setElement(main.oneeSama.article(this.model.attributes));
		return this;
	},
	insertIntoDOM() {
		main.$threads.children('#' + this.model.get('op'))
			.children('blockquote, .omit, form, article[id]:last')
			.last()
			.after(this.$el);
		this.autoExpandImage().fun();
	},
	renderEditing(model, editing) {
		this.$el.toggleClass('editing', !!editing);
		if (!editing)
			this.$el.children('blockquote')[0].normalize();
	}
});

// Extend with common mixins
_.extend(Article.prototype, postCommon);
