// Client entry point

import { loadFromDB, page, posts, storeMine } from './state'
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
import { checkBottom, getCookie, deleteCookie } from "./util"
import assignHandlers from "./client"
import initModeration from "./mod"

// Load all stateful modules in dependency order
async function start() {
	const frag = document.getElementById("threads")
	extractConfigs()

	await open()
	if (page.thread) {
		await loadFromDB(page.thread)

		// Add a stored thread OP, made by the client to "mine"
		const addMine = getCookie("addMine")
		if (addMine) {
			const id = parseInt(addMine)
			storeMine(id, id)
			deleteCookie("addMine")
		}
	}

	initOptions()

	if (page.thread) {
		renderThread()
	} else {
		await renderBoard()
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

	// Load auxiliary modules
	initUI()
	initModeration()
}

start().catch(err => {
	alert(err.message)
	throw err
})
