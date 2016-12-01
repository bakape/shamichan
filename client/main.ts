// Client entry point

import { displayLoading, loadFromDB, page } from './state'
import { start as connect } from './connection'
import { open } from './db'
import { render as renderBoard } from './page/board'
import renderThread from "./page/thread"
import { extractConfigs } from "./page/common"
import { exec, init } from './defer'
import { loadModule } from "./util"
import { checkBottom, scrollToAnchor } from "./scroll"
import { setTitle } from "./tab"

// Load all stateful modules in dependency order
async function start() {
	const frag = document.getElementById("threads")
	extractConfigs(frag)

	await open()
	await loadFromDB()
	init()

	if (page.thread) {
		renderThread(frag, false)
	} else {
		renderBoard(frag, false)
	}

	scrollToAnchor()
	checkBottom()
	connect()
	exec()
	setTitle(frag.querySelector("#page-title").textContent)
	displayLoading(false)

	// Load auxiliary modules
	const modules = [
		"mod/login", "etc", "hover", "posts/posting/drop", "options/loop",
		"keyboard", "posts/menu", "page/boardNavigation",
	]
	for (let m of modules) {
		loadModule(m)
	}
}

start().catch(err => {
	alert(err.message)
	throw err
})
