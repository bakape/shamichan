'use strict';

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

// Set the theme, if setting exists
(function () {
	var options;
	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {};

	var link = document.getElementById('theme'),
		m = link.href.match(/^(.*\/)[^\/]+?\.css$/),
		theme = options.theme;
	if (m && theme)
		link.href = m[1] + hotConfig.css[theme + '.css'];
})();
