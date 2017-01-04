// Client entry point

import { displayLoading, loadFromDB, page, posts } from './state'
import { start as connect } from './connection'
import { open } from './db'
import { initOptions } from "./options"
import initPosts from "./posts"
import { Post } from "./posts"
import { ThreadData } from "./common"
import {
	renderBoard, extractConfigs, setThreadTitle, renderThread
} from './page'
import { default as initUI, setTitle } from "./ui"
import { checkBottom } from "./util"
import assignHandlers from "./client"
import initModeration from "./mod"

// Load all stateful modules in dependency order
async function start() {
	const frag = document.getElementById("threads")
	extractConfigs()
	initOptions()

	await open()
	await loadFromDB()

	if (page.thread) {
		renderThread("")
	} else {
		renderBoard()
	}

	checkBottom()
	connect()
	assignHandlers()
	initPosts()

	if (page.thread) {
		setThreadTitle(posts.get(page.thread) as Post & ThreadData)
	} else {
		setTitle(frag.querySelector("#page-title").textContent)
	}

	displayLoading(false)


	// Load auxiliary modules
	initUI()
	initModeration()
}

start().catch(err => {
	alert(err.message)
	throw err
})
