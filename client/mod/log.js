/*
Moderation log modal view
 */

const main = require('main'),
	{Backbone, common, dispatcher, oneeSama, lang, modals} = main;

const ModLogView = Backbone.View.extend({
	className: 'modal mod panel',
	initialize() {
		this.$el.appendTo(main.$overlay);
		modals.log = this;

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
		delete modals.log;
		this.remove();
	}
});
module.exports = ModLogView;
