var BOARD, THREAD, BUMP, PAGE;

(function () {
	var p = location.pathname;
	BOARD = p.match(/^\/(.+?)\//)[1];
	var t = p.match(/\/(\d+)$/);
	THREAD = t ? parseInt(t[1], 10) : 0;
	BUMP = !!p.match(/\/live$/);
	PAGE = p.match(/\/page(\d+)$/);
	PAGE = PAGE ? parseInt(PAGE[1], 10) : -1;
})();
