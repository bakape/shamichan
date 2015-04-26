// Empty placeholder function
dispatcher[DEF.RADIO] = function() {};

(function() {
	if (!config.RADIO || isMobile)
		return;

	// Extend banner functionality with r/a/dio stream info rendering
	Banner.__proto__.renderRadio = function(data) {
		data = JSON.parse(data);
		this.$el.children('#banner_center').html(
			'<a href="http://r-a-d.io/" target="_blank">'
			+ '[' + data.listeners + '] ' + data.dj
			+ '</a>&nbsp;&nbsp;<a title="Click to google song"'
			+ 'href="https://google.com/search?q='
			+ encodeURIComponent(data.np) + '" target="_blank"><b>'
			+ data.np + '</b></a>'
		);
	};

	Banner.__proto__.clearRadio = function() {
		this.$el.children('#banner_center').empty();
	};

	dispatcher[DEF.RADIO] = function(msg) {
		// R/a/dio banner is disabled on mobile
		if (!options.get('nowPlaying'))
			Banner.renderRadio(msg[0]);
	};
})();