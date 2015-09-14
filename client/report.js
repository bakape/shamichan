/*
 Report posts you don't like
 */

const main = require('./main'),
	{$, $script, _, Backbone, common, lang} = main;

// TODO: Rewrite this and move to API v2

const pubKey = main.config.RECAPTCHA_PUBLIC_KEY,
	captchaTimeout = 5 * 60 * 1000,
	repLang = lang.reports,
	reports = {};
let panel;

let Report = Backbone.Model.extend({
	defaults: {
		status: 'setup',
		hideAfter: true
	},
	request_new() {
		Recaptcha.create(pubKey, 'captcha', {
			theme: 'clean',
			callback: () => this.set('status', 'ready')
		});

		if (this.get('timeout'))
			clearTimeout(this.get('timeout'));

		this.set('timeout', setTimeout(() => {
			this.set('timeout', 0);
			this.request_new();
		}, captchaTimeout));
	},
	did_report() {
		delete reports[this.id];
		if (this.get('timeout')) {
			clearTimeout(this.get('timeout'));
			this.set('timeout', 0);
		}

		setTimeout(() => this.trigger('destroy'), 1500);

		if (this.get('hideAfter'))
			this.get('post').set('hide', true);
	}
});

var ReportPanel = Backbone.View.extend({
	id: 'report-panel',
	tagName: 'form',
	className: 'modal',
	events: {
		submit: 'submit',
		'click .close': 'remove',
		'click .hideAfter': 'hide_after_changed'
	},
	initialize() {
		this.$captcha = $('<div id="captcha"/>');
		this.$message = $('<div class="message"/>');
		this.$submit = $('<input>', {
			type: 'submit',
			val: 'Report'
		});
		let $hideAfter = $('<input>', {
			class: 'hideAfter',
			type: 'checkbox',
			checked: this.model.get('hideAfter')
		});
		let $hideLabel = $('<label>and hide</label>').append($hideAfter);

		const num = this.model.get('post').get('num');

		this.$el.append(
			repLang.post + ' ',
			main.oneeSama.postRef(num).safe,
			'<a class="close" href="#">x</a>',
			this.$message,
			this.$captcha,
			this.$submit,
			' ',
			$hideLabel
		);

		/* HACK */
		if (window.x_csrf) {
			this.model.set('hideAfter', false);
			$hideLabel.remove();
		}

		this.listenTo(this.model, {
			'change:error': this.error_changed,
			'change:status': this.status_changed,
			destroy: this.remove
		});
	},
	render() {
		this.error_changed();
		this.status_changed();
		return this;
	},
	submit() {
		if (this.model.get('status') != 'ready')
			return false;
		main.send([
			common.REPORT_POST,
			parseInt(this.model.get('post').get('num'), 10),
			Recaptcha.get_challenge(),
			Recaptcha.get_response()
		]);
		this.model.set('status', 'reporting');
		return false;
	},
	error_changed() {
		this.$message.text(this.model.get('error'));
	},
	status_changed() {
		const status = this.model.get('status');
		this.$submit
			.prop('disabled', status != 'ready')
			.toggle(status !== 'done')
			.val(status === 'reporting' ? repLang.reporting : lang.report);
		this.$captcha.toggle(
			_.contains(['ready', 'reporting', 'error'], status)
		);
		if (status === 'done')
			this.$('label').remove();

		let msg;
		if (status === 'done')
			msg = repLang.submitted;
		else if (status == 'setup')
			msg = repLang.setup;
		else if (status == 'error'
			|| (status == 'ready' && this.model.get('error')))
				msg = 'E';

		this.$message.text(msg === 'E' ? this.model.get('error') : msg);
		this.$message.toggle(!!msg).toggleClass('error', msg == 'E');

		// not strictly view logic, but only relevant when visible
		if (status == 'ready')
			this.focus();
		else if (status == 'done')
			this.model.did_report();
		else if (status == 'error')
			this.model.request_new();
	},
	hide_after_changed(e) {
		this.model.set('hideAfter', e.target.checked);
	},
	focus() {
		Recaptcha.focus_response_field();
	},
	remove() {
		Backbone.View.prototype.remove.call(this);
		if (panel === this) {
			panel = null;
			Recaptcha.destroy();
		}
		return false;
	}
});

main.reply('report', function(post) {
	const url = 'https://www.google.com/recaptcha/api/js/recaptcha_ajax.js';
	$script(url, function () {
		const num = post.get('num');
		let model = reports[num];
		if (!model) {
			reports[num] = model = new Report({
				id: num,
				post: post
			});
		}
		if (panel) {
			if (panel.model === model) {
				panel.focus();
				return;
			}
			panel.remove();
		}
		panel = new ReportPanel({model: model});
		panel.render().$el.appendTo('body');
		if (window.Recaptcha)
			model.request_new();
		else {
			model.set({
				status: 'error',
				error: repLang.loadError
			});
		}
	});
});

main.dispatcher[common.REPORT_POST] = function(msg) {
	const report = reports[msg[0]];
	if (report)
		report.set(msg[1] || {status: 'done'});
};
