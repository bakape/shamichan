(function() {
	// Set the theme
	var theme = localStorage.theme || config.defaultCSS
	document.getElementById('theme').href = '/ass/css/' + theme + '.css'

	window.lang = localStorage.lang || config.lang.default
})()
