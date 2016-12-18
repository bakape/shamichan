// Client entry point

import { displayLoading, loadFromDB, page, posts } from './state'
import { start as connect } from './connection'
import { open } from './db'
import { render as renderBoard } from './page/board'
import renderThread, { setThreadTitle } from "./page/thread"
import { extractConfigs } from "./page/common"
import { exec, init } from './defer'
import { loadModule } from "./util"
import { checkBottom, scrollToAnchor } from "./scroll"
import { ThreadData } from "./posts/models"
import { setTitle } from "./tab"
import { Post } from "./posts/models"

// Load all stateful modules in dependency order
async function start() {
	const frag = document.getElementById("threads")
	extractConfigs()

	await open()
	await loadFromDB()
	init()

	if (page.thread) {
		renderThread("")
	} else {
		renderBoard()
	}

	scrollToAnchor()
	checkBottom()
	connect()
	exec()

	if (page.thread) {
		setThreadTitle(posts.get(page.thread) as Post & ThreadData)
	} else {
		setTitle(frag.querySelector("#page-title").textContent)
	}

	displayLoading(false)

	// Load auxiliary modules
	const modules = [
		"mod/login", "etc", "hover", "posts/posting/drop",
		"posts/posting/threadCreation", "options/loop", "keyboard",
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
