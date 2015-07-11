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
		let specs = this.specs = [
			'clearSelection',
			'spoilerImages',
			'deleteImages',
			'deletePosts',
			'lockThread',
			'toggleMnemonics'
		];
		if (ident.auth === 'Admin')
			specs.push('sendNotification', 'dispatchFun', 'renderPanel');

		let controls = '<span>';
		for (let i = 0; i < specs.length; i++) {
			const ln = lang.mod[specs[i]];
			controls += common.parseHTML
				`<a class="modButton" data-kind="${i}" title="${ln[1]}">
					${ln[0]}
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
		this[this.specs[event.target.getAttribute('data-kind')]]();
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
