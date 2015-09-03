/*
View containing controls of moderation actions
 */

const main = require('main'),
	input = require('./input'),
	panels = require('./panels'),
	util = require('./util'),
	{$, _, Backbone, common, lang, modals} = main,
	{parseHTML} = common;

const childViews = {
	ban: input.ban,
	log: require('./panels').log,
	adminPanel: panels.adminPanel,
	notification: input.notification
};

const ToolboxView = Backbone.View.extend({
	id: 'toolbox',
	className: 'mod modal panel',
	initialize() {
		this.render();
	},
	render() {
		const specs = this.specs = [
			'clearSelection',
			'spoilerImages',
			'deleteImages',
			'deletePosts',
			'modLog'
		];
		if (common.checkAuth('moderator', main.ident)) {
			specs.push('toggleMnemonics', 'lockThreads', 'ban');
			if (common.checkAuth('admin', main.ident))
				specs.push('sendNotification', 'renderPanel');
		}

		let controls = '<span>';
		for (let i = 0; i < specs.length; i++) {
			const ln = lang.mod[specs[i]];
			controls += parseHTML
				`<a class="modButton" data-kind="${i}" title="${ln[1]}">
					${ln[0]}
				</a>`;
		}
		controls += '</span>';
		this.$controls = $(controls);

		this.$checkboxToggle = $(parseHTML
				`<style>
				.postCheckbox {
					display: inline-block;
				}
			</style>`
		)
			.appendTo('head')
			// Disabled only works if the emelemnt is in the DOM
			.prop('disabled', true);

		this.$toggle = $(`<a id="toolboxToggle">${lang.show}</a>`);
		this.$el.prepend(this.$controls, this.$toggle)
			.appendTo(main.$overlay);

		// Sets mnemonic visbility
		this.$mnemonicStyle = $(parseHTML
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
		this[this.specs[event.target.getAttribute('data-kind')]](event);
	},
	clearSelection() {
		util.loopCheckboxes(el => el.checked = false);
	},
	toggleMnemonics() {
		const hide = localStorage.noMnemonics === 'true';
		this.$mnemonicStyle.prop('disabled', hide);
		localStorage.noMnemonics = !hide;
	},
	send(type) {
		main.send([common[type], ...util.getSelected()]);
	},
	spoilerImages() {
		this.send('SPOILER_IMAGES');
	},
	deleteImages() {
		this.send('DELETE_IMAGES');
	},
	toggleChild(type) {
		if(modals[type])
			modals[type].kill();
		else
			new childViews[type];
	},
	// Push a notification message to all clients
	sendNotification() {
		this.toggleChild('notification');
	},
	modLog() {
		this.toggleChild('log');
	},
	ban() {
		this.toggleChild('ban');
	},
	renderPanel() {
		this.toggleChild('adminPanel');
	},
	deletePosts() {
		this.send('DELETE_POSTS');
	},
	lockThreads() {
		for (let num of util.getSelected()) {
			const model = main.state.posts.get(num);
			// Model exists and is an OP
			if (!model || model.get('op'))
				continue;
			main.send([
				common[!model.get('locked') ? 'LOCK_THREAD' : 'UNLOCK_THREAD'],
				num
			]);
		}
	}
});

main.$toolbox = new ToolboxView({
	model: new Backbone.Model()
});
