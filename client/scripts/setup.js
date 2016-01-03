(function() {
	// Set the theme
	var theme = localStorage.theme || config.defaultCSS
	document.getElementById('theme').href = config.hard.HTTP.media + 'css/'
		+ theme + '.css?v=' + clientHash

	window.lang = localStorage.lang || config.lang.default
})()
