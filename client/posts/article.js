/*
 * Reply posts
 */

const main = require('../main'),
	postCommon = require('./common'),
	{_, Backbone} = main;

const Article = module.exports = Backbone.View.extend({
	tagName: 'article',
	render() {
		this.setElement(main.oneeSama.article(this.model.attributes));
		return this;
	},
	insertIntoDOM() {
		const last = _.last(document.query('#p' + this.model.get('op'))
			.queryAll('article[id]'));
		last.after(this.el);
		this.autoExpandImage().fun();
	}
});

// Extend with common mixins
_.extend(Article.prototype, postCommon);
