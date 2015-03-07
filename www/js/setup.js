var BOARD, THREAD, BUMP, PAGE, mediaURL, options, themeVersion;
// NOTE: options gets turned into a backbone model later

(function () {
	var p = location.pathname;
	BOARD = p.match(/^\/(.+?)\//)[1];
	var t = p.match(/\/(\d+)$/);
	THREAD = t ? parseInt(t[1], 10) : 0;
	BUMP = /\/$/.test(p);
	t = p.match(/\/page(\d+)$/);
	PAGE = t ? parseInt(t[1], 10) : -1;

	if (!mediaURL) {
		var sc = document.getElementsByTagName('script');
		for (var i = 0; i < sc.length; i++) {
			var m = /^(.*)js\/setup.js\?v=\d+$/.exec(sc[i].src);
			if (m) {
				mediaURL = m[1];
				break;
			}
		}
	}
	if (!mediaURL)
		alert("Couldn't determine mediaURL! Hardcode it.");

	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {};

	// Set theme to match options
	var link = document.getElementById('theme'),
		m = link.href.match(/^(.*\/)[^\/]+?\.css$/),
		theme = options['board.'+BOARD+'.theme'];
	if (m && theme)
		link.href = m[1] + hotConfig.css[theme + '.css'];
})();
