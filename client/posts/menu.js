/*
 Post action header dropdown menu
 */

let main = require('../main'),
	{$, _, Backbone, common, lang} = main;

let MenuView = module.exports = Backbone.View.extend({
	// Maping of menu items to their handler message bus commands
	actions: {
		report: 'report',
		hide: 'hide'
	},
	events: {
		click: 'handleClick'
	},
	initialize(args) {
		this.render(args.parent);
	},
	render(parent) {
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
	handleClick(e) {
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
