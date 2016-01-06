/*
 * Loads the dependancies in order and aggregates exports from various modules
 */

// NOTE: The entire bundle uses strict mode through the Babel transpiler

/*
 * Because we are going to attach listeners to these all over the place, have
 * to load soome core modules in specific order. Also avoids nasty circular
 * dependancy, by placing some of the exports here and not in child modules.
 */

// DOM level 4 polyfill
import * as dom4 from '../vendor/dom4'
import * as Cookie from '../vendor/js-cookie'
import {parseEl, parseHTML} from './util'

/*
 Ofload expensive and not that neccessary initialisation logic till
 after the core modules are started
 */
const deferred = []

/**
 * Add a function to be executed, once the module finishes loading
 * @param {function} func
 */
export function defer(func) {
    deferred.push(func)
}

/**
 * Execute all stored deferred functions
 */
export function execDeferred() {
	while (deferred.length > 0) {
		deferred.shift()()
	}
}

// Configuration object, passed from the server
export const config = window.config
config.mediaURL = config.hard.HTTP.media // Shorthand

// Hash of the the configuration object
export const configHash = window.configHash

// Combined hash of the current client-side files. Used for transparent
// versioning.
export const clientHash = window.clientHash

// Indicates, if in mobile mode. Determined server-side.
export const isMobile = window.isMobile

// Cached DOM elements
export const $threads = document.query('threads')
export const $name = document.query('#name')
export const $email = document.query('#email')
export const $banner = document.query('#banner')

// Clear cookies, if versions mismatch.
const cookieVersion = 3
if (localStorage.cookieVersion != cookieVersion) {
	for (let cookie in Cookie.get()) {
		// Clear legacy cookies that were set for each board separatly.
		// Otherwise, they would override the new ones.
		const paths = config.boards.enabled.slice()
		paths.push('', '/')
		for (let path of paths) {
			Cookie.remove(cookie, {path})
		}
	}
	localStorage.cookieVersion = cookieVersion
}

// You can invoke the client-side debug mode with the `debug=true` query string
if (/[&\?]debug=true/.test(location.href)) {
	config.hard.debug = true
}
/*
if (config.hard.debug) {
	radio.DEBUG = true
	window.Backbone = Backbone // Export Backbone instance for easier debugging
	radio.tuneIn('main') // Log all channel traffic
}
*/

// Load language-specific CSS
document.head.appendChild(parseEl(parseHTML
	`<style>
		.locked:after {
			content: "${lang.thread_locked}";
		}
		.locked > header nav:after {
			content: " (${lang.locked})";
		}
	</style>`))

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

/*
// 2nd tier dependacy modules. These are needed before the websocket
// connection is opened, because they populate the dispatcher handler object.
extend(main, {
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
extend(main, {
	history: require('./history'),
	hide: require('./hide')
})
*/

execDeferred()
events.request('loading:hide')
