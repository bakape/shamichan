// Set the theme
(function () {
	var options;
	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) {}
	var theme = (options && options.theme)
		? options.theme
		: hotConfig.BOARD_CSS[location.href.match(/\/([a-zA-Z0-9]+?)\//)[1]];
	document.getElementById('theme').href = config.MEDIA_URL + 'css/'
		+ hotConfig.css[theme + '.css'];
})();

// Check for browser compatibility by trying to detect some ES6 features
(function() {
	if (localStorage.browserChecked)
		return;
	if (typeof Set !== 'function'
		|| 	typeof WeakSet !== 'function'
		|| typeof Number.isNaN !== 'function'
	) {
		alert('Your browser appears to be outdated. In case you experience '
			+ 'problems with browsing meguca, please consider installing the '
			+ 'latest version of either Google Chrome, Mozilla Firefox, Opera,'
			+ ' Microsoft Edge or any of the mobile versions of these browsers.'
			+ 'This message will not appear again.'
		);
	}
	localStorage.setItem('browserChecked', true);
})();
