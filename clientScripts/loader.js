/*
Selects and loads the client files
 */

(function () {
	// Check for browser compatibility by trying to detect some ES6 features
	function check(func) {
		try {
			return eval('(function(){' + func + '})();')
		}
		catch(e) {
			return false
		}
	}

	var tests = [
		// Arrow functions
		'return (()=>5)()===5;',
		// Block scopped const
		'"use strict";  const bar = 123; {const bar = 456;} return bar===123;',
		// Block-scoped let
		'"use strict"; let bar = 123;{ let bar = 456; }return bar === 123;',
		// Computed object properties
		"var x='y';return ({ [x]: 1 }).y === 1;",
		// Shorthand object properties
		"var a=7,b=8,c={a,b};return c.a===7 && c.b===8;",
		// Template strings
		'var a = "ba"; return `foo bar${a + "z"}` === "foo barbaz";',
		// for...of
		'var arr = [5]; for (var item of arr) return item === 5;',
		// Spread operator
		'return Math.max(...[1, 2, 3]) === 3',
		// Class statement
		'"use strict"; class C {}; return typeof C === "function"',
		// Super call
		'"use strict"; var passed = false;'
			+ 'class B {constructor(a) {  passed = (a === "barbaz")}};'
			+ 'class C extends B {constructor(a) {super("bar" + a)}};'
			+ 'new C("baz"); return passed;'
	]
	var legacy
	for (var i = 0; i < tests.length; i++) {
		if (!check(tests[i])) {
			// Load client with full ES5 compliance
			legacy = true
			break
		}
	}

	// TODO: Load core-js/es6, if above tests fail

	var mediaURL = config.hard.HTTP.media,
		$script = require('scriptjs')

	window.loadModule = function (file, cb) {
		var url = mediaURL + 'js/es' + (legacy ? "5" : "6") + "/" + file
			+ ".js?v=" + clientHash
		$script(url, cb)
	}

	window.loadDep = function (file, cb) {
	    $script(mediaURL + "js/vendor/" + file + ".js?v=" + clientHash, cb)
	}

	$script(mediaURL + 'js/lang/' + lang + '.js?v=' + clientHash, function () {
		loadModule("main")
	})
})()
