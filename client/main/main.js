/*
 * Loads the depandancies in order and aggregates exports from various modules
 */

// NOTE: The entire bundle uses strict mode through the Babel transpiler

/*
 * Because we are going to attach listeners to these all over the place, have
 * to load soome core modules in specific order. Also avoids nasty circular
 * dependancy, by placing some of the exports here and not in child modules.
 */
const _ = require('underscore'),
	Backbone = require('backbone'),
	Cookie = require('js-cookie'),
	radio = require('backbone.radio')

import FSM from './fsm'

// Load standard library ES6 polyfills
require('core-js')
// Load DOM level 4 polyfill
require('dom4')
// Load Backbone.View native JS replacement
require('backbone.nativeview')
Backbone.View = Backbone.NativeView

// Central aplication object and message bus
let main = module.exports = radio.channel('main')

_.extend(main, {
	// Bind dependancies to main object for pretier destructuring requires
	_, Backbone, Cookie, FSM,
	$script: require('scriptjs'),
	SockJS: require('sockjs-client'),

	/*
	 Ofload expensive and not that neccessary initialisation logic till
	 after the core modules are started
	 */
	_deferred: [],
	defer(func) {
		main._deferred.push(func)
		return main
	},
	execDefered() {
		for (let func of this._deferred) {
			func()
		}
	},

	// Websocket call handler map. Store them here, to avoid requiring
	// modules in the wrong order.
	dispatcher: {}
})

// Import configuration variables from the template HTML
_.extend(main, {config, configHash, clientHash, isMobile})

// Clear cookies, if versions mismatch. Get regenerated each client start
// anyway.
const cookieVersion = 3
if (localStorage.cookieVersion != cookieVersion) {
	for (let cookie in Cookie.get()) {

		// Clear legacy cookies that were set for each board separatly.
		// Otherwise, they would override the new ones.
		const paths = main.config.boards.enabled.slice()
		paths.push('')
		for (let path of paths) {
			Cookie.remove(cookie, {path})
		}
	}
	localStorage.cookieVersion = cookieVersion
}

// You can invoke the client-side debug mode with the `debug=true` query string
if (/[&\?]debug=true/.test(location.href))
	main.config.hard.debug = true
if (main.config.hard.debug) {
	radio.DEBUG = true
	window.Backbone = Backbone // Export Backbone instance for easier debugging
	radio.tuneIn('main') // Log all channel traffic
}

main.send = main.request.bind(main, 'send') // Shorthand

/*
 Core modules. The other will be more or less decoupled, but these are the
 monolithic foundation.
 */
main.Memory = require('./memory')
const lang = main.lang = require('lang'),
	state = main.state = require('./state'),
	util = main.util = require('./util')

/*
// Initialise main rendering object
let oneeSama = main.oneeSama = new common.OneeSama({
	op: state.page.get('thread'),
	lang,
	// Core post link handler
	tamashii(num) {
		let frag;
		const op = state.links[num];
		if (op) {
			const desc = num in state.mine.readAll() && this.lang.you;
			frag = this.postRef(num, op, desc);
		}
		else
			frag = '>>' + num;
		return frag;
	}
});
*/
main.options = require('./options')
main.scroll = require('./scroll')
state.page.set('tabID', common.randomID(32))

// Load language-specific CSS
document.head.appendChild(util.parseDOM(common.parseHTML
	`<style>
		.locked:after {
			content: "${lang.thread_locked}";
		}
		.locked > header nav:after {
			content: " (${lang.locked})";
		}
	</style>`));

_.extend(main, {
	// Cached DOM elements
	$threads: document.query('threads'),
	$name: document.query('input[name=name]'),
	$email: document.query('input[name=email]'),

	connSM: new FSM('load'),
	postSM: new FSM('none')
});

// 2nd tier dependacy modules. These are needed before the websocket
// connection is opened, because they populate the dispatcher handler object.
_.extend(main, {
	loop: require('./loop'),
	time: require('./time'),
	amusement: require('./amusement')
});

// Load post models and views
main.posts = require('./posts')
main.Extract = require('./extract')
// Start the client
main.client = require('./client')
main.conection = require('./connection')

// Load independant auxilary modules
_.extend(main, {
	history: require('./history'),
	hide: require('./hide')
})

main.execDefered()
main.request('loading:hide')
