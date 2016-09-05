// Client entry point

// TODO: Remove, when proper structure done
import * as client from './client'
let c = client  // Prevents the compiler from removing as an unused import
c = null

import {displayLoading, loadFromDB, page} from './state'
import {initTemplates} from "./render"
import {start as connect} from './connection'
import {open} from './db'
import loadPage from './page/load'
import BoardNavigation from './page/boardNavigation'
import {exec} from './defer'
import bindThreadCreation from './posts/posting/threadCreation'
import {initOptions} from './options'
import bindEtc from './etc'
import bindOptionsListeners from "./options/loop"
import bindShortcuts from "./keyboard"

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

	initTemplates()
	initOptions()
	await open()
	await loadFromDB()
	renderPage()
	new BoardNavigation()
	bindThreadCreation()
	bindEtc()
	bindOptionsListeners()
	bindShortcuts()
	exec()
	await pageLoader
	connect()
	displayLoading(false)
}

start()
