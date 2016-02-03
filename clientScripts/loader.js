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

	// Load all client modules as precompiled System.register format modules
	var meta = {}
	meta['es5/*'] = meta['es6/*'] = {format: 'register'}

	System.config({
		baseURL: '/ass/js',
		defaultJSExtensions: true,
		// Alias the appropriate language pack to "lang"
		map: {
			lang: 'lang/' + (localStorage.lang || config.lang.default),
			underscore: 'vendor/underscore',
			'js-cookie': 'vendor/js-cookie'
		},
		meta: meta
	})

	// Load core-js polyfill, if above tests fail
	if (legacy) {
		System.import('vendor/corejs').then(loadMain)
	} else {
		loadMain()
	}

	// Application entry point
	function loadMain() {
		System.import((legacy ? 'es5' : 'es6') + '/main')
	}
})()
