(function() {
	// Prevent from loading stale pages on browser state resume
	if (Date.now() - renderTime >= 60000)
		return location.reload();

	var options;
	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) {}

	// Set the theme
	var mediaURL = config.MEDIA_URL;
	var theme = (options && options.theme) ? options.theme
		: hotConfig.DEFAULT_CSS;
	document.getElementById('theme').href = mediaURL + 'css/' + theme
		+ '.css?v=' + cssHash;
})();
