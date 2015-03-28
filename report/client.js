(function () {
	var pubKey = reportConfig.RECAPTCHA_PUBLIC_KEY;
	var captchaTimeout = 5 * 60 * 1000;
	var REPORTS = {};
	var PANEL;

	if (pubKey)
		menuOptions.push('Report');

	var Report = Backbone.Model.extend({
		defaults: {
			status: 'setup',
			hideAfter: true,
		},

		request_new: function () {
			var self = this;

			Recaptcha.create(pubKey, 'captcha', {
				theme: 'clean',
				callback: function () {
					self.set('status', 'ready');
				}
			});

			if (this.get('timeout'))
				clearTimeout(this.get('timeout'));

			this.set('timeout', setTimeout(function () {
				self.set('timeout', 0);
				self.request_new();
			}, captchaTimeout));
		},

		did_report: function () {
			delete REPORTS[this.id];
			if (this.get('timeout')) {
				clearTimeout(this.get('timeout'));
				this.set('timeout', 0);
			}

			var self = this;
			setTimeout(function () {
				self.trigger('destroy');
			}, 1500);

			if (this.get('hideAfter'))
				this.get('post').set('hide', true);
		},
	});

	var ReportPanel = Backbone.View.extend({
		id: 'report-panel',
		tagName: 'form',
		className: 'modal',

		events: {
			submit: 'submit',
			'click .close': 'remove',
			'click .hideAfter': 'hide_after_changed',
		},

		initialize: function () {
			this.$captcha = $('<div id="captcha"/>');
			this.$message = $('<div class="message"/>');
			this.$submit = $('<input>', {type: 'submit', val: 'Report'});
			var $hideAfter = $('<input>', {
				class: 'hideAfter',
				type: 'checkbox',
				checked: this.model.get('hideAfter'),
			});
			var $hideLabel = $('<label>and hide</label>')
				.append($hideAfter);

			var num = this.model.id;

			this.$el
			.append('Reporting post ')
			.append($('<a/>', {href: '#'+num, text: '>>'+num}))
			.append('<a class="close" href="#">x</a>')
			.append(this.$message)
			.append(this.$captcha)
			.append(this.$submit)
			.append(' ', $hideLabel);

			/* HACK */
			if (window.x_csrf) {
				this.model.set('hideAfter', false);
				$hideLabel.remove();
			}

			this.listenTo(this.model, {
				'change:error': this.error_changed,
				'change:status': this.status_changed,
				destroy: this.remove,
			});
		},

		render: function () {
			this.error_changed();
			this.status_changed();
			return this;
		},

		submit: function () {
			if (this.model.get('status') != 'ready')
				return false;
			send([DEF.REPORT_POST, this.model.id, Recaptcha.get_challenge(),
					Recaptcha.get_response()]);
			this.model.set('status', 'reporting');
			return false;
		},

		error_changed: function () {
			this.$message.text(this.model.get('error'));
		},

		status_changed: function () {
			var status = this.model.get('status');
			this.$submit
				.prop('disabled', status != 'ready')
				.toggle(status != 'done')
				.val(status=='reporting' ? 'Reporting...' : 'Report');
			this.$captcha.toggle(
				_.contains(['ready', 'reporting', 'error'], status));
			if (status == 'done')
				this.$('label').remove();

			var msg;
			if (status == 'done')
				msg = 'Report submitted!';
			else if (status == 'setup')
				msg = 'Obtaining reCAPTCHA...';
			else if (status == 'error')
				msg = 'E';
			else if (status == 'ready' && this.model.get('error'))
				msg = 'E';
			this.$message.text(msg=='E' ? this.model.get('error') : msg);
			this.$message
				.toggle(!!msg)
				.toggleClass('error', msg == 'E');

			// not strictly view logic, but only relevant when visible
			if (status == 'ready')
				this.focus();
			else if (status == 'done')
				this.model.did_report();
			else if (status == 'error')
				this.model.request_new();
		},

		hide_after_changed: function (e) {
			this.model.set('hideAfter', e.target.checked);
		},

		focus: function () {
			Recaptcha.focus_response_field();
		},

		remove: function () {
			Backbone.View.prototype.remove.call(this);
			if (PANEL == this) {
				PANEL = null;
				Recaptcha.destroy();
			}
			return false;
		},
	});

	var ajaxJs = 'https://www.google.com/recaptcha/api/js/recaptcha_ajax.js';

	menuHandlers.Report = function (post) {
		var num = post.id;
		var model = REPORTS[num];
		if (!model)
			REPORTS[num] = model = new Report({id: num, post: post});

		if (PANEL) {
			if (PANEL.model === model) {
				PANEL.focus();
				return;
			}
			PANEL.remove();
		}
		PANEL = new ReportPanel({model: model});
		PANEL.render().$el.appendTo('body');
		yepnope({load: ajaxJs, callback: function () {
			if (window.Recaptcha)
				model.request_new();
			else
				model.set({
					status: 'error',
					error: "Couldn't load reCATPCHA.",
				});
		}});
	};

	dispatcher[DEF.REPORT_POST] = function (msg, op) {
		var num = msg[0], etc = msg[1];
		var report = REPORTS[num];
		if (report)
			report.set(msg[1] || {status: 'done'});
	};
})();
