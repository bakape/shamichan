// Client entry point

import { displayLoading, loadFromDB, page } from './state'
import { initTemplates } from "./render"
import { start as connect } from './connection'
import { open } from './db'
import loadPage from './page/common'
import { exec, init } from './defer'
import { loadModule } from "./util"
import { checkBottom, scrollToAnchor } from "./scroll"

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
	await pageLoader
	scrollToAnchor()
	checkBottom()
	connect()
	exec()
	displayLoading(false)

	// Load auxiliary modules
	const modules = [
		"etc", "hover", "posts/posting/drop", "options/loop", "keyboard",
		"posts/menu", "page/boardNavigation",
	]
	for (let m of modules) {
		loadModule(m)
	}
}

start().catch(err =>
	alert(err.message))
