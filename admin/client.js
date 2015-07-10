/*
Client-side administration logic
 */

let	main = require('main'),
	{$, $threads, _, Backbone, common, config, etc, lang} = main;

const ident = window.IDENT;

// Pass login status to ./www/js/login.js
window.loggedInUser = ident.email;
window.x_csrf = ident.csrf;

$('<link/>', {
	rel: 'stylesheet',
	href: `${config.MEDIA_URL}css/mod.css?v=${cssHash}`
}).appendTo('head');

let ToolboxView = Backbone.View.extend({
	tagName: 'div',
	id: 'toolbox',
	className: 'mod modal',
	initialize() {
		this.render();
	},
	render() {
		let order = ['clear', 7, 8, 9, 11, 'mnemonics'];
		let specs = this.specs = {
			clear: ['Clear', 'clearSelection'],
			7: ['Spoiler', 'spoilerImages'],
			8: ['Del Img', 'deleteImages'],
			9: ['Del Post', 'deletePosts'],
			11: ['Lock', 'lockThread'],
			mnemonics: ['Mnemonics', 'toggleMnemonics']
		};
		if (ident.auth === 'Admin') {
			order.push('notification', 'fun', 'panel');
			_.extend(specs, {
				notification: ['Notification', 'sendNotification'],
				fun: ['Fun', 'dispatchFun'],
				panel: ['Panel', 'renderPanel']
			});
		}

		let controls = '<span>';
		for (let kind of order) {
			controls += common.parseHTML
				`<a class="modButton" data-kind="${kind}">
					${specs[kind][0]}
				</a>`;
		}
		controls += '</span>';
		this.$controls = $(controls);

		this.$checkboxToggle = $('<style/>', {
			html: '.postCheckbox {display: inline-block;}'
		})
			.appendTo('head')
			// Disabled only works if the emelemnt is in the DOM
			.prop('disabled', true);

		this.$toggle = $(`<a id="toolboxToggle">${lang.show}</a>`);
		this.$el.prepend(this.$controls, this.$toggle)
			.appendTo('body');

		// Sets mnemonic visability
		this.$mnemonicStyle = $(common.parseHTML
			`<style>
				header > .mod.addr {
					display: none;
				}
			</style>`
		)
			.appendTo('head')
			.prop('disabled', localStorage.noMnemonics !== 'true');
		return this;
	},
	events: {
		'click #toolboxToggle': 'toggleButtons',
		'click .modButton': 'buttonHandler'
	},
	toggleButtons() {
		const hidden = !this.model.get('shown');
		this.$toggle.text(lang[hidden ? 'hide' : 'show']);
		this.$controls.toggle(0);
		this.$checkboxToggle.prop('disabled', !hidden);
		this.model.set('shown', hidden);
	},
	buttonHandler(event) {
		this[this.specs[event.target.getAttribute('data-kind')][1]]();
	},
	getSelected() {
		let checked = [];
		this.loopCheckboxes(function (el) {
			if (el.checked)
				checked.push(etc.getID(el));
		});

		// Postform will not have an ID, so we remove falsy values
		return _.compact(checked);
	},
	clearSelection() {
		this.loopCheckboxes(el => el.checked = false);
	},
	loopCheckboxes(func) {
		const els = $threads[0].getElementsByClassName('postCheckbox');
		for (let i = 0; i < els.length; i++) {
			func(els[i]);
		}
	},
	toggleMnemonics() {
		const hide = localStorage.noMnemonics === 'true';
		this.$mnemonicStyle.prop('disabled', hide);
		localStorage.noMnemonics = !hide;
	},
	spoilerImages() {
		main.command('send', [common.SPOILER_IMAGES, ...this.getSelected()]);
	}
});

let toolbox = new ToolboxView({
	model: new Backbone.Model()
});
