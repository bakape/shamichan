// Client entry point

import { displayLoading, loadFromDB, page } from './state'
import { initTemplates } from "./render"
import { start as connect } from './connection'
import { open } from './db'
import { render as renderBoard } from './page/board'
import { exec, init } from './defer'
import { loadModule } from "./util"
import { checkBottom, scrollToAnchor } from "./scroll"

// Load all stateful modules in dependency order
async function start() {
	initTemplates()
	await open()
	await loadFromDB()
	init()
	if (!page.thread) {
		const frag = document.getElementById("threads")
		renderBoard(frag)
	}
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

start().catch(err => {
	alert(err.message)
	throw err
})
