/*
Moderation log modal view
 */

const main = require('main'),
	{Backbone, common, dispatcher, oneeSama, lang, modals} = main;

const PanelView = Backbone.View.extend({
	className: 'modal mod panel',
	initialize() {
		this.$el.appendTo(main.$overlay);
		modals[this.type] = this;

		// Register websocket handler
		dispatcher[this.kind] = msg => this.render(msg[0]);
		// Send request
		main.send([this.kind]);
	},
	render(info) {
		if (!info.length) {
			return this.el.innerHTML = "God's in his Heaven. All's right"
				+ " with the world.";
		}
		this.el.innerHTML = this.renderContents(info);

		// Scroll to the end of the log
		this.el.scrollTop = this.el.scrollHeight;
		return this;
	},
	kill() {
		delete dispatcher[this.kind];
		delete modals[this.type];
		this.remove();
	}
});

const ModLogView = PanelView.extend({
	type: 'log',
	kind: common.MOD_LOG,
	renderContents(info) {
		return table(info, act => [
			// Unbans do not have a target post
			act.num ? oneeSama.postRef(act.num, act.op).safe : '',
			lang.mod.formatLog(act),
			oneeSama.time(act.time)
		]);
	}
});
exports.log = ModLogView;

const AdminPanelView = PanelView.extend({
	type: 'adminPanel',
	id: 'adminPanel',
	kind: common.ADMIN_PANEL,
	events: {
		'click .unban': 'unban'
	},
	renderContents(info) {
		this.banCount = 0;
		return table(info, ban => {
			this.banCount++;
			return [
				oneeSama.mnemonic(ban[0]),
				oneeSama.time(ban[1]),
				`<a class="unban" data-id="${ban[0]}">${lang.mod.unban}</a>`
			];
		});
	},
	unban(event) {
		const el = event.target;
		main.send([common.UNBAN, el.getAttribute('data-id')]);
		el.parentElement.parentElement.remove();

		// Check if any bans left
		if (!--this.banCount)
			this.kill();
	}
});
exports.adminPanel = AdminPanelView;

function table(rows, func) {
	let html = '<table>';
	for (let row of rows) {
		html += '<tr>';
		for (let cell of func(row)) {
			html += `<td>${cell}</td>`;
		}
		html += '</tr>';
	}
	html += '</table>';
	return html;
}
