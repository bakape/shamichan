(function() {
	var options;
	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) {}

	// Set the theme
	var mediaURL = imports.config.MEDIA_URL;
	var theme = (options && options.theme) ? options.theme
		: imports.hotConfig.DEFAULT_CSS;
	document.getElementById('theme').href = mediaURL + 'css/' + theme
		+ '.css?v=' + imports.cssHash;
})();
