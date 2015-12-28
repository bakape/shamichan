(function() {
	var options
	try {
		options = JSON.parse(localStorage.options)
	}
	catch (e) {}

	// Set the theme
	var mediaURL = config.hard.HTTP.media
	var theme = (options && options.theme) ? options.theme : config.defaultCSS
	document.getElementById('theme').href = mediaURL + 'css/' + theme
		+ '.css?v=' + clientHash

	window.lang = (options && options.lang) || config.lang.default 
})()
