// TODO: Rewrite all banner-related code with Backbone and move them here

// Notification messages bellow the banner
var NotificationView = Backbone.View.extend({
	initialize: function(msg){
		this.render(msg);
	},

	events: {
		'click': 'remove'
	},

	render: function(msg){
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

dispatcher[DEF.NOTIFICATION] = function(msg){
	new NotificationView(msg[0]);
};


dispatcher[DEF.RADIO] = function(msg) {
	// R/a/dio banner is disabled on mobile
	if (isMobile || options.get('nowPlaying'))
		return;
	Banner.renderRadio(msg[0]);
};

dispatcher[DEF.UPDATE_BANNER] = function(msg) {
	Banner.renderInfo(msg[0]);
};

var BannerView = Backbone.View.extend({
	initialize: function() {

	},

	renderRadio: function(data) {
		data = JSON.parse(data);
		this.$el.children('#banner_center').html(
			'<a href="http://r-a-d.io/" target="_blank">' +
				'[' + data.listeners + '] ' + data.dj +
			'</a>&nbsp;&nbsp;<a title="Click to google song" href="https://google.com/search?q=' +
				encodeURIComponent(data.np) + '" target="_blank"><b>' + data.np + '</b></a>'
		);
	},

	renderInfo: function(msg) {
		this.$el.children('#banner_info').html(msg);
	},

	clearRadio: function() {
		this.$el.children('#banner_center').empty();
	}
});

var Banner = new BannerView({
	el: $('#banner')[0]
});