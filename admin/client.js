/*
Client-side administration logic
 */

const main = require('main'),
	{$, $threads, _, Backbone, common, config, dispatcher, etc, lang,
		oneeSama, state} = main,
	{parseHTML} = common;

// Only used to affect some client rendering practises. Anything actually
// needing security has stricter authorisation checks.
const ident = main.ident = window.IDENT;

// Pass login status to ./www/js/login.js
window.loggedInUser = ident.email;
window.x_csrf = ident.csrf;

$('<link/>', {
	rel: 'stylesheet',
	href: `${config.MEDIA_URL}css/mod.css?v=${cssHash}`
}).appendTo('head');

{
	// Add staff board to board navigation
	const staff = config.STAFF_BOARD;
	$('#navTop')
		.children('a')
		.last()
		.after(` / <a href="../${staff}/" class="history">${staff}</a>`);
}

// Container for the overlay
let $overlay = $('<div id="modOverlay"></div>').appendTo('body');

let ToolboxView = Backbone.View.extend({
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
			'toggleMnemonics',
			'modLog'
		];
		if (common.checkAuth('moderator', ident)) {
			specs.push('lockThreads');
			if (common.checkAuth('admin', ident)) {
				specs.push('sendNotification'
					/* Useless for now , 'renderPanel'*/);
			}
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
			.appendTo($overlay);

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
	getSelected() {
		const checked = [];
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
	send(type) {
		main.send([common[type], ...this.getSelected()]);
	},
	spoilerImages() {
		this.send('SPOILER_IMAGES');
	},
	deleteImages() {
		this.send('DELETE_IMAGES');
	},
	// Push a notification message to all clients
	sendNotification() {
		let box = this.notificationBox;
		if (box) {
			this.notificationBox = null;
			return box.remove();
		}

		let self = this;
		this.notificationBox = new InputBoxView({
			fields: ['msg'],
			handler(msg) {
				self.notificationBox = null;
				main.send([common.NOTIFICATION, msg[0]]);
			}
		});
	},
	modLog() {
		if (!this.logPanel)
			this.logPanel = new ModLogView();
		else {
			this.logPanel.kill();
			this.logPanel = null;
		}
	},
	deletePosts() {
		this.send('DELETE_POSTS');
	},
	lockThreads() {
		for (let num of this.getSelected()) {
			const model = state.posts.get(num);
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

let toolbox = new ToolboxView({
	model: new Backbone.Model()
});

// Input box character sizes
const sizeMap = {
	msg: 20
};

let InputBoxView = Backbone.View.extend({
	className: 'mod inputBox',
	events: {
		submit: 'submit'
	},
	initialize(args) {
		this.handler = args.handler;
		this.render(args);
	},
	render(args) {
		let html = '<form>';
		for (let id of args.fields) {
			html += parseHTML `<input ${{
				type: 'text',
				'data-id': id,
				size: sizeMap[id],
				placeholder: lang.mod.placeholders[id]
			}}>`;
		}
		html += parseHTML
			`<input type="submit" value="${lang.send}">
			</form>`;
		this.$el
			.html(html)
			.prependTo(toolbox.$el)
			.find('input').first().focus();
	},
	submit(event) {
		event.preventDefault();
		let values = [];
		$(event.target).children('input[type=text]').each(function () {
			values.push(this.value);
		});
		this.handler(values);
		this.remove();
	}
});

// Scrollable message log
let ModLogView = Backbone.View.extend({
	className: 'modal mod panel',
	initialize() {
		this.$el.appendTo($overlay);

		// Register websocket handler
		dispatcher[common.MOD_LOG] = msg => this.render(msg[0]);
		// Request moderation log
		main.send([common.MOD_LOG]);
	},
	render(info) {
		if (!info.length) {
			return this.el.innerHTML = "God's in his Heaven. All's right"
				+ " with the world.";
		}
		let html = '<table>';
		for (let act of info) {
			html += '<tr>';
			const cells = [
				oneeSama.postRef(act.num, act.op).safe,
				lang.mod.formatLog(act),
				oneeSama.time(act.time)
			];
			for (let cell of cells) {
				html += `<td>${cell}</td>`;
			}
			html += '</tr>';
		}
		html += '</table>';
		this.el.innerHTML = html;

		// Scroll to the end of the log
		this.el.scrollTop = this.el.scrollHeight;
		return this;
	},
	kill() {
		delete dispatcher[common.MOD_LOG];
		this.remove();
	}
});

// Display title on post
main.$name.after(parseHTML
	`<label title="${lang.mod.title[1]}" class="mod">
		<input type="checkbox" id="authName">
		 ${lang.mod.title[0]}
	 </label>`);
const $authName = $('#authName');

oneeSama.hook('fillMyName', function ($el) {
	const checked = $authName[0].checked;
	$el.toggleClass(ident.auth === 'admin' ? 'admin' : 'moderator', checked);
	if (checked)
		$el.append(' ## ' + state.hotConfig.get('staff_aliases')[ident.auth]);
});
$authName.change(() => main.request('postForm:indentity'));

// Extend default allocation request
override(main.posts.posting.ComposerView.prototype, 'allocationMessage',
	function (orig, ...args) {
		const msg = orig.call(this, ...args);
		if ($authName[0].checked)
			msg.auth = ident.auth;
		return msg;
	});

function override(parent, method, upgrade) {
	const orig = parent[method];
	parent[method] = function (...args) {
		return upgrade.call(this, orig, ...args);
	}
}
