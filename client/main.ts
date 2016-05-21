/*
  Client entry point.
  NOTE: All modules use strict mode implicitly
 */

 // TODO: Remove, when proper structure done
import * as options from './options'
import * as client from './client'
const o = options
const c = client

import {displayLoading} from './state'
import {exec} from './defer'
import {start as connect} from './connection'
import {loadFromDB} from './state'
import {open} from './db'

// Clear cookies, if versions mismatch.
const cookieVersion = 4
if (localStorage.getItem("cookieVersion") !== cookieVersion.toString()) {
	for (let cookie of document.cookie.split(";")) {
		const eqPos = cookie.indexOf("="),
			name = eqPos > -1 ? cookie.substr(0, eqPos) : cookie
		document.cookie = name + "=;expires=Thu, 01 Jan 1970 00:00:00 GMT"
	}
	localStorage.setItem("cookieVersion", cookieVersion.toString())
}

// Load all stateful modules in dependancy order
async function start() {
	await open()
	await loadFromDB()
	connect()
	exec()
	displayLoading(false)
}

start()
