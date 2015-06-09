/*
 * Handles all things banner and notifications
 */

let main = require('./main'),
	{$, Backbone, common, dispatcher, options} = main;

const modalMap = {
	'options': 'options-panel',
	'banner_identity': 'identity',
	'banner_FAQ': 'FAQ',
	'banner_schedule': 'schedule'
};

let BannerView = Backbone.View.extend({
	initialize() {
		this.$center = this.$el.children('#banner_center');
		this.$info = this.$el.children('#banner_info');
		// Publish a listener to the message bus
		main.comply('banner:radio:clear', this.clearRadio, this);
	},
	events: {
		'click .bfloat': 'revealBmodal'
	},
	renderInfo(msg) {
		this.$info.html(msg);
	},
	// Toggle the display of the modal windows under the banner
	revealBmodal(event) {
		let $target = $(event.target).closest('.bfloat');
		const bmodal = modalMap[$target.attr('id')];
		if (!bmodal)
			return;
		let $el = $('#' + bmodal);
		const isShown = $el.is(':visible');
		$('.bmodal').hide();
		// We hid the currently displayed window. All is well
		if (isShown)
			return;
		// Place 5 pixels bellow banner
		$el.css('top', $('#banner').outerHeight() + 5 + 'px');
		$el.show();
	},
	// r/a/dio stream info rendering
	renderRadio(data) {
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
	clearRadio() {
		this.$center.empty();
	}
});

let banner = exports.view = new BannerView({
	el: document.getElementById('banner')
});

// Notification messages bellow the banner
let NotificationView = exports.notification = Backbone.View.extend({
	initialize(msg) {
		this.render(msg);
	},
	events: {
		'click': 'remove'
	},
	render(msg) {
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

main.comply('notification', msg => new NotificationView(msg));

dispatcher[common.NOTIFICATION] = msg => new NotificationView(msg[0]);
dispatcher[common.UPDATE_BANNER] = msg => banner.renderInfo(msg[0]);
dispatcher[common.RADIO] = function(msg) {
	// R/a/dio banner is disabled on mobile
	if (options.get('nowPlaying') && !main.isMobile)
		banner.renderRadio(msg[0]);
};
