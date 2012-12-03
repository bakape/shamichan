var BOARD, THREAD, BUMP, PAGE, options;

(function () {
	var p = location.pathname;
	BOARD = p.match(/^\/(.+?)\//)[1];
	var t = p.match(/\/(\d+)$/);
	THREAD = t ? parseInt(t[1], 10) : 0;
	BUMP = /\/$/.test(p);
	t = p.match(/\/page(\d+)$/);
	PAGE = t ? parseInt(t[1], 10) : -1;

	try {
		options = JSON.parse(localStorage.options);
	}
	catch (e) { }
	if (!options)
		options = {};

	var theme = options['board.'+BOARD+'.theme'];
	if (theme) {
		var link = document.getElementById('theme');
		var m = link.href.match(/^(.*\/)[^\/]+(-v\d+)\.css$/);
		if (m)
			link.href = m[1] + theme + m[2] + '.css';
	}
})();
