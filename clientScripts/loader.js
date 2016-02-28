/*
 Selects and loads the client files
 Use only pure ES5.
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
			+ 'new C("baz"); return passed;',
		// Promises
		'return typeof Promise === "function"',
		// ServiceWorker
		'return typeof navigator.serviceWorker === "object"',
		// Default parameters
		'return (function (a = 1, b = 2) { return a === 3 && b === 2; }(3));',
		// Destructuring decliration
		'var [a,,[b],c] = [5,null,[6]];return a===5 && b===6 && c===undefined',
		// Parameter destructuring
		'return function([a,,[b],c]){return a===5 && b===6 && c===undefined;}'
			+ '([5,null,[6]])'
	]

	for (var i = 0; i < tests.length; i++) {
		if (!check(tests[i])) {
			alert("Browser outdated. Install latest Chrome/Firefox/Opera")
			return
		}
	}

	initModuleLoader()

	navigator.serviceWorker.register("/worker.js").then(function () {
		return System.import("vendor/dom4")
	}).then(function () {
		// Wait until serviceWorker is ready
		return navigator.serviceWorker.ready
	}).then(function () {
		return System.import('client/main')
	})
})()
