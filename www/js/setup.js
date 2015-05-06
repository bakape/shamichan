// Set the theme. This needs to be done as fast as possible
(function () {
	var BOARD;
	BOARD = location.pathname.match(/^\/(.+?)\//)[1];
	var options;
	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {};

	var link = document.getElementById('theme'),
		m = link.href.match(/^(.*\/)[^\/]+?\.css$/),
		theme = options['board.'+BOARD+'.theme'];
	if (m && theme)
		link.href = m[1] + hotConfig.css[theme + '.css'];
})();
