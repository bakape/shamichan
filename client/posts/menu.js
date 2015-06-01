/*
 Post action header menu
 */

let $ = require('jquery'),
	_ = require('underscore'),
	Backbone = require('backbone'),
	main = require('../main'),
	common = main.common,
	lang = main.lang;

let MenuView = module.exports = Backbone.View.extend({
	// Maping of menu items to their handler message bus commands
	actions: {
		report: 'report',
		hide: 'hide'
	},

	events: {
		click: 'handleClick'
	},

	initialize: function(args) {
		this.render(args.parent);
	},

	render: function(parent) {
		let html = '<ul class="popup-menu">';
		for (let action in this.actions) {
			html += `<li data-type="${action}">${lang[action]}</li>`
		}
		html += '</ul>';
		this.setElement(html);
		this.$el.appendTo(parent);

		// Remove view, if clicked outside
		setTimeout(() => {
			main.$doc.one('click', e => {
				if (!$(e.target).closest('ul').is(this.$el))
					this.remove();
			})
		}, 300);
	},

	// Forward post model to appropriate handler
	handleClick: function(e) {
		e.stopPropagation();
		main.command(this.actions[e.target.getAttribute('data-type')],
			this.model
		);
		this.remove();
	}
});

main.comply('menu:extend', action =>
	_.extend(MenuView.prototype.actions, action)
);
