/*
 * Handles all things banner and notifications
 */

var $ = require('jquery'),
	Backbone = require('backbone'),
	main = require('./main'),
	common = main.common,
	options = main.options;

// Notification messages bellow the banner
var NotificationView = exports.notification = Backbone.View.extend({
	initialize: function(msg) {
		this.render(msg);
	},

	events: {
		'click': 'remove'
	},

	render: function(msg) {
		$('.notification').remove();
		let $banner = banner.$el;
		let $el = $(common.parseHTML
			`<span class="notification modal"
				style="top: ${$banner.outerHeight() + 5 + 'px'};"
			>
				<b class="admin">
					${msg}
				</b>
			</span>`
		)
			.insertAfter($banner);
		this.setElement($el[0]);
		return this;
	}
});

main.dispatcher[common.NOTIFICATION] = function(msg) {
	new NotificationView(msg[0]);
};

main.comply('notification', msg => new NotificationView(msg));

main.dispatcher[common.UPDATE_BANNER] = function(msg) {
	banner.renderInfo(msg[0]);
};

var BannerView = Backbone.View.extend({
	initialize: function() {
		this.$center = this.$el.children('#banner_center');
		this.$info = this.$el.children('#banner_info');
		// Publish a listener to the message bus
		main.comply('clearRadioBanner', this.clearRadio, this);
	},

	events: {
		'click .bfloat': 'revealBmodal'
	},

	renderInfo: function(msg) {
		this.$info.html(msg);
	},

	// Toggle the display of the modal windows under the banner
	revealBmodal: function(event) {
		var $target = $(event.target).closest('.bfloat'),
			id = $target.attr('id'),
			bmodal;
		if (id == 'options')
			bmodal = 'options-panel';
		else if (id == 'banner_identity')
			bmodal = 'identity';
		else if (id == 'banner_FAQ')
			bmodal = 'FAQ';
		else if (id == 'banner_schedule')
			bmodal = 'schedule';

		if (!bmodal)
			return;
		var $el = $('#' + bmodal),
			isShown = $el.is(':visible');
		$('.bmodal').hide();
		// We hid the currently displayed window. All is well
		if (isShown)
			return;
		// Place 5 pixels bellow banner
		$el.css('top', $('#banner').outerHeight() + 5 + 'px');
		$el.show();
	},

	// r/a/dio stream info rendering
	renderRadio: function(data) {
		data = JSON.parse(data);
		this.$center.html(common.parseHTML
			`<a href="http://r-a-d.io/" target="_blank">
				[${data.listeners}] ${data.dj}
			</a>
			&nbsp;&nbsp;
			<a title="Click to google song"
				href="https://google.com/search?q=${encodeURIComponent(data.np)}"
				target="_blank"
			>
				<b>${data.np}</b>
			</a>`
		);
	},

	clearRadio: function() {
		this.$center.empty();
	}
});

var banner = exports.view = new BannerView({
	el: document.getElementById('banner')
});

main.dispatcher[common.RADIO] = function(msg) {
	// R/a/dio banner is disabled on mobile
	if (options.get('nowPlaying') && !main.isMobile)
		banner.renderRadio(msg[0]);
};
