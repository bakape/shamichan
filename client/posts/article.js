/*
 * Reply posts
 */

let main = require('../main'),
	postCommon = require('./common'),
	{$, _, Backbone, options, state} = main;

var Article = module.exports = Backbone.View.extend({
	tagName: 'article',
	initialize() {
		/*
		 * XXX: A bit ineficient, because first an empty element is renderred
		 * and then a proper one.
		 *
		 * An element is not empty only on postForms and extraction
		 */
		if (!this.el.innerHTML)
			this.render().insertIntoDOM();
		this.initCommon();
	},
	render() {
		// Pass this model's links to oneeSama for renderring
		main.oneeSama.links = this.model.get('links');
		this.setElement(main.oneeSama.article(this.model.attributes));
		return this;
	},
	insertIntoDOM() {
		main.$threads.children('#' + this.model.get('op'))
			.children('blockquote, .omit, form, article[id]:last')
			.last()
			.after(this.$el);
		this.autoExpandImage();
	},
	renderEditing(model, editing) {
		this.$el.toggleClass('editing', !!editing);
		if (!editing)
			this.$el.children('blockquote')[0].normalize();
	}
});

// Extend with common mixins
_.extend(Article.prototype, postCommon);
