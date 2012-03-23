var BOARD, THREAD, BUMP, PAGE, options;

(function () {
	var p = location.pathname;
	BOARD = p.match(/^\/(.+?)\//)[1];
	var t = p.match(/\/(\d+)$/);
	THREAD = t ? parseInt(t[1], 10) : 0;
	BUMP = !!p.match(/\/live$/);
	PAGE = p.match(/\/page(\d+)$/);
	PAGE = PAGE ? parseInt(PAGE[1], 10) : -1;

	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {};

	var theme = options['board.'+BOARD+'.theme'];
	if (theme) {

		// TEMP MIGRATION
		var m = theme.match(/(.*)-v\d+$/);
		if (m) {
			theme = m[1];
			if (theme == 'meta')
				theme = 'mawaru';
			options['board.'+BOARD+'.theme'] = theme;
			try {
				localStorage.options = JSON.stringify(options);
			}
			catch (e) { }
		}
		// END MIGRATION

		var link = document.getElementById('theme');
		var m = link.href.match(/^(.*\/)[^\/]+(-v\d+)\.css$/);
		if (m)
			link.href = m[1] + theme + m[2] + '.css';
	}
})();
