/*
Selects and loads the client files
 */

(function() {
	if (!localStorage.legacyChecked) {
		// Check for browser compatibility by trying to detect some ES6 features
		if (typeof Set !== 'function'
			|| 	typeof WeakSet !== 'function'
			|| typeof Number.isNaN !== 'function'
			|| !checkConst()
			|| !checkTemplateStrings()
		) {
			// Load client with full ES5 complience
			localStorage.legacy = true;
		}
		localStorage.legacyChecked = true;
	}

	var $script = require('scriptjs'),
		base = config.MEDIA_URL + 'js/',
		end = '.js?v=' + clientHash;

	$script(base + 'lang/' + lang + end, function() {
		var client = localStorage.legacy ? 'legacy' : 'client';
		$script(base + client + end);
	});

	function checkConst() {
		try {
			return eval('(function(){"use strict";const foo=123;return'
				+ ' foo===123;})();'
			);
		}
		catch(e) {
			return false;
		}
	}

	function checkTemplateStrings() {
		try {
			return eval('(function(){var a = "ba";'
				+'return `foo bar${a + "z"}` === "foo barbaz";})();'
			);
		}
		catch(e) {
			return false;
		}
	}
})();
