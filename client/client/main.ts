/*
 * Client entry point.
 * NOTE: All modules use strict mode through the Babel transpiler
 */

import {connect} from './connection'

// Clear cookies, if versions mismatch.
const cookieVersion = 4
if (localStorage.getItem("cookieVersion") != cookieVersion) {
	for (let cookie of document.cookie.split(";")) {
		const eqPos = cookie.indexOf("="),
			name = eqPos > -1 ? cookie.substr(0, eqPos) : cookie
		document.cookie = name + "=;expires=Thu, 01 Jan 1970 00:00:00 GMT"
	}
	localStorage.setItem("cookieVersion", cookieVersion.toString())
}

connect()

// // Load language-specific CSS
// document.head.appendChild(parseEl(parseHTML
// 	`<style>
// 		.locked:after {
// 			content: "${lang.thread_locked}";
// 		}
// 		.locked > header nav:after {
// 			content: " (${lang.locked})";
// 		}
// 	</style>`))

//events.request('loading:hide')
