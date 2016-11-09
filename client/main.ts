// Client entry point

import { displayLoading, loadFromDB, page, isMobile } from './state'
import { initTemplates } from "./render"
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
import { checkBottom, scrollToAnchor } from "./scroll"
import bindMenu from "./posts/menu"

// Load all stateful modules in dependency order
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
	scrollToAnchor()
	checkBottom()
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

start().catch(err =>
	alert(err.message))
