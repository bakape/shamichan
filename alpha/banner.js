/*
 * Handles all things banner and notifications
 */

var $ = require('jquery'),
	Backbone = require('backbone'),
	common = require('../common'),
	main = require('./main');

// Notification messages bellow the banner
var NotificationView = exports.notification = Backbone.View.extend({
	initialize: function(msg) {
		this.render(msg);
	},
	events: {
		'click': 'remove'
	},
	render: function(msg) {
		var $banner = $('#banner');
		$('.notification').remove();
		this.$el = $('<span/>', {
			class: 'notification modal'
		})
			.html('<b class="admin">' + msg + '</b>')
			.css('top', $banner.outerHeight() + 5 + 'px')
			.insertAfter($banner);
		return this;
	}
});

main.dispatcher[common.NOTIFICATION] = function(msg) {
	new NotificationView(msg[0]);
};

main.dispatcher[common.UPDATE_BANNER] = function(msg) {
	banner.renderInfo(msg[0]);
};

var BannerView = Backbone.View.extend({
	initialize: function() {
		this.setElement(document.getElementById('banner'));
	},
	events: {
		'click .bfloat': 'revealBmodal'
	},
	renderInfo: function(msg) {
		this.$el.children('#banner_info').html(msg);
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
		this.$el.children('#banner_center').html(
			'<a href="http://r-a-d.io/" target="_blank">'
			+ '[' + data.listeners + '] ' + data.dj
			+ '</a>&nbsp;&nbsp;<a title="Click to google song"'
			+ 'href="https://google.com/search?q='
			+ encodeURIComponent(data.np) + '" target="_blank"><b>'
			+ data.np + '</b></a>'
		);
	},
	clearRadio: function() {
		this.$el.children('#banner_center').empty();
	}
});

var banner = exports.view = new BannerView();

main.dispatcher[common.RADIO] = function(msg) {
	// R/a/dio banner is disabled on mobile
	if (!options.get('nowPlaying') || main.isMobile)
		banner.renderRadio(msg[0]);
};