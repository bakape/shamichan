// Client entry point

import { displayLoading, loadFromDB, page, isMobile } from './state'
import { initTemplates, read } from "./render"
import { start as connect } from './connection'
import { open } from './db'
import loadPage from './page/common'
import BoardNavigation from './page/boardNavigation'
import { exec, init } from './defer'
import bindThreadCreation from './posts/posting/threadCreation'
import bindEtc from './etc'
import bindOptionsListeners from "./options/loop"
import bindShortcuts from "./keyboard"
import { loadModule } from "./util"
import { checkBottom } from "./scroll"
import bindMenu from "./posts/menu"

// Clear cookies, if versions mismatch
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
	await open()
	await loadFromDB()
	init()
	renderPage()
	new BoardNavigation()
	bindThreadCreation()
	bindEtc()
	bindOptionsListeners()
	bindShortcuts()
	bindMenu()
	await pageLoader
	read(() =>
		checkBottom())
	connect()
	exec()
	displayLoading(false)

	// Conditionally load desktop-only modules
	if (!isMobile) {
		await Promise.all([
			loadModule("hover"),
			loadModule("posts/posting/drop"),
		])
	}
}

start()
