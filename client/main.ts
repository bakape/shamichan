/*
  Client entry point.
  NOTE: All modules use strict mode implicitly
 */

 // TODO: Remove, when proper structure done
import * as options from './options'
const o = options

import {displayLoading} from './state'
import {exec} from './defer'
import {start} from './connection'

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

start()
exec()
displayLoading(false)
