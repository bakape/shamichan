/*
 * Reply posts
 */

const main = require('../main'),
	PostCommon = require('./common'),
	{_, Backbone} = main;

module.exports = PostCommon.extend({
	tagName: 'article',
	render() {
		this.setElement(main.oneeSama.article(this.model.attributes));
		return this;
	},
	insertIntoDOM() {
		_.last(document.query('#p' + this.model.get('op')).queryAll('article'))
			.after(this.el);
		this.autoExpandImage().fun();
	}
});
