/*
 * Client entry point.
 * NOTE: All modules use strict mode through the Babel transpiler
 */

import * as dom4 from '../vendor/dom4' // DOM level 4 polyfill
import * as Cookie from '../vendor/js-cookie'
import {parseEl, parseHTML} from './util'
import {defer, execDeferred} from './defer'
import lang from 'lang'

// TEMP: Will later get imported by the post modules
import * as state from './state'

import OptionsPanel from './options/view'

//Renders the options panel, after more important computation has been done
defer(() => new OptionsPanel())

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
	config.debug = true
}

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
//events.request('loading:hide')
