// Set the theme. This needs to be done as fast as possible
(function () {
	'use strict';

	let options;
	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {};

	let link = document.getElementById('theme');
	const m = link.href.match(/^(.*\/)[^\/]+?\.css$/),
		theme = options.theme;
	if (m && theme)
		link.href = m[1] + hotConfig.css[theme + '.css'];
})();
