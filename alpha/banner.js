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
			'class': 'notification modal'
		})
			.html('<b class="admin">' + msg + '</b>')
			.css('top', $banner.outerHeight() + 5 + 'px')
			.insertAfter($banner);
		return this;
	}
});

main.dispatcher[common.NOTIFICATION] = function(msg){
	new NotificationView(msg[0]);
};

main.dispatcher[common.UPDATE_BANNER] = function(msg) {
	Banner.renderInfo(msg[0]);
};

var BannerView = Backbone.View.extend({
	initialize: function() {},

	renderInfo: function(msg) {
		this.$el.children('#banner_info').html(msg);
	},
});

var Banner = exports.view = new BannerView({
	el: $('#banner')[0]
});