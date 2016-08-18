// Client entry point

 // TODO: Remove, when proper structure done
import * as client from './client'
let c = client  // Prevents the compiler from removing as an unused import
c = null

import {displayLoading} from './state'
import {start as connect} from './connection'
import {loadFromDB, page} from './state'
import {open} from './db'
import loadPage from './page/load'
import BoardNavigation from './page/boardNavigation'
import {exec} from './defer'
import bindThreadCreation from './posts/posting/threadCreation'
import {initOptions} from './options'
import bindEtc from './etc'

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
	// Load asynchronously and concurently as fast as possible
	let renderPage: () => void
	const ready = new Promise<void>((resolve) =>
		renderPage = resolve)
	const pageLoader = loadPage(page, ready)

	initOptions()
	await open()
	await loadFromDB()
	renderPage()
	new BoardNavigation()
	connect()
	bindThreadCreation()
	bindEtc()
	exec()
	await pageLoader
	displayLoading(false)
}

start()
