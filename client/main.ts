// Client entry point

import { loadFromDB, page, posts, storeMine } from './state'
import { start as connect, connSM, connState } from './connection'
import { open } from './db'
import { initOptions } from "./options"
import initPosts from "./posts"
import { postSM, postEvent, FormModel } from "./posts"
import { renderBoard, extractConfigs, renderThread } from './page'
import initUI from "./ui"
import {
	checkBottom, getCookie, deleteCookie, trigger, scrollToBottom,
} from "./util"
import assignHandlers from "./client"
import initModeration from "./mod"
import { persistMessages } from "./options"

// Load all stateful modules in dependency order
async function start() {
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

	// Check for legacy options and remap
	const oldNowPlaying = localStorage.getItem("nowPlaying")
	if (oldNowPlaying === "true") {
		localStorage.setItem("nowPlaying", "r/a/dio")
	} else if (oldNowPlaying === "false") {
		localStorage.setItem("nowPlaying", "none")
	}

	initOptions()

	if (page.thread) {
		renderThread()

		// Open a cross-thread quoting reply
		connSM.once(connState.synced, () => {
			const data = localStorage.getItem("openQuote")
			if (!data) {
				return
			}
			const i = data.indexOf(":"),
				id = parseInt(data.slice(0, i)),
				sel = data.slice(i + 1)
			localStorage.removeItem("openQuote")
			if (posts.get(id)) {
				postSM.feed(postEvent.open);
				(trigger("getPostModel") as FormModel).addReference(id, sel)
				requestAnimationFrame(scrollToBottom)
			}
		})

		persistMessages()
		connect()
		checkBottom()
		assignHandlers()
	} else {
		await renderBoard()
	}

	initPosts()
	initUI()
	initModeration()
}

start().catch(err => {
	alert(err.message)
	throw err
})
