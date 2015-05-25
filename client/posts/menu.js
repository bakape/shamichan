/*
 Post action header menu
 */

let _ = require('underscore'),
	Backbone = require('backbone'),
	main = require('../main'),
	common = main.common,
	lang = main.lang;

module.exports = Backbone.View.extend({
	// Maping of menu items to their handler message bus commands
	actions: {
		focus: 'scroll:focus',
		report: 'report',
		hide: 'hide'
	},

	events: {
		click: 'handleClick'
	},

	initialize: function(args) {
		this.render(args.parent);
		main.comply('menu:extend', action => _.extend(this.actions, action));
	},

	render: function(parent) {
		let html = '<ul class="popup-menu">';
		for (let action in this.actions) {
			html += `<li data-type="${action}">${lang[action]}</li>`
		}
		html += '</ul>';
		this.setElement(html);
		this.$el.appendTo(parent);
	},

	handleClick: function(e) {
		e.stopPropagation();
		main.command(this.actions[e.target.getAttribute('data-type')]);
		this.remove();
	}
});
