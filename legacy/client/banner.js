/*
 * Handles all things banner and notifications
 */

// Notification messages bellow the banner
const NotificationView = exports.notification = Backbone.View.extend({
	initialize(msg) {
		this.render(msg);
	},
	events: {
		'click': 'remove'
	},
	render(msg) {
		for (let el of document.queryAll('.notification')) {
			el.remove();
		}
		const attrs = {
			class: 'notification modal',
			style: `top: ${banner.el.offsetHeight + 5}px;`
		};
		const el = etc.parseDOM(common.parseHTML
			`<span ${attrs}>
				<b class="admin">
					${msg}
				</b>
			</span>`);
		banner.el.after(el);
		this.setElement(el);
		return this;
	}
});
main.reply('notification', msg => new NotificationView(msg));

dispatcher[common.NOTIFICATION] = msg => new NotificationView(msg[0]);
