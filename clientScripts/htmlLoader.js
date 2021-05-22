(function() {
	function loadScript(path) {
		var head = document.getElementsByTagName('head')[0];
		var script = document.createElement('script');
		script.type = 'text/javascript';
		script.src = '/assets/' + path + '.js';
		head.appendChild(script);
		return script;
	}

	loadScript("js/html/main").onload = function () {
		require("html/main");
	};
})();
