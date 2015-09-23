(function() {
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
