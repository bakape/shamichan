/*
Various input boxes
 */

const main = require('main'),
	util = require('./util'),
	{$, Backbone, common, lang, modals} = main,
	{parseHTML} = common;

const InputBoxView = Backbone.View.extend({
	className: 'mod inputBox',
	events: {
		submit: 'submit'
	},
	initialize() {
		modals[this.type] = this;
		this.render();
	},
	render() {
		const html = parseHTML
			`<form>
				${this.renderInput()}
				<input type="submit" value="${lang.send}">
			</form>`;
		this.$el
			.html(html)
			.prependTo(main.$toolbox.$el)
			.find('input').first().focus();
	},
	submit(event) {
		event.preventDefault();
		const values = [];
		$(event.target).find('input[type!=submit]').each(function () {
			values.push(this.type === 'checkbox' ? this.checked : this.value);
		});
		this.handler(values);
	},
	kill() {
		delete modals[this.type];
		this.remove();
	}
});

const NotificationInputView = InputBoxView.extend({
	type: 'notification',
	renderInput() {
		return parseHTML `<input ${{
			type: 'text',
			size: 25,
			placeholder: lang.mod.placeholders.msg
		}}>`;
	},
	handler(msg) {
		main.send([common.NOTIFICATION, msg[0]]);
		this.kill();
	}
});
exports.notification = NotificationInputView;

const BanInputView = InputBoxView.extend({
	type: 'ban',
	renderInput() {
		let html = '';
		for (let field of ['days', 'hours', 'minutes']) {
			html += parseHTML `<input ${{
				type: 'number',
				placeholder: lang.mod.placeholders[field]
			}}>`
		}
		html += parseHTML `<input ${{
			type: 'text',
			size: 25,
			placeholder: lang.mod.placeholders.reason
		}}>`;
		const [label, title] = lang.mod.displayBan;
		html += parseHTML
			`<label ${{title}}>
				<input type="checkbox">
				${label}
			</label>`;
		return html;
	},
	handler(info) {
		// Ensure reason field is filled
		if (!info[3])
			return this.renderReasonPrompt();
		
		// Coerce time units and checkbox value to integers
		for (let i = 0; i < 3; i++) {
			info[i] = +info[i];
		}
		info[4] = +info[4];
		for (let num of util.getSelected()) {
			main.send([common.BAN, num, ...info]);
		}
		this.kill();
	},
	renderReasonPrompt() {
		this.$el.find('.reasonPrompt').remove();
		this.$el
			.append(parseHTML
				`<b class="reasonPrompt admin">
					${lang.mod.needReason}
				</b>`)
			.find('input[type=text]').focus();
	}
});
exports.ban = BanInputView;
