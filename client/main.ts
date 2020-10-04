// Client entry point

import { loadFromDB, page, posts, storeMine, storeSeenPost } from './state'
import { start as connect, connSM, connState } from './connection'
import { open } from './db'
import { initOptions } from "./options"
import initPosts from "./posts"
import { postSM, postEvent, FormModel } from "./posts"
import {
	renderBoard, extractConfigs, renderThread, init as initPage
} from './page'
import * as thread from "./page/thread";
import initUI from "./ui"
import {
	checkBottom, getCookie, deleteCookie, trigger, scrollToBottom,
} from "./util"
import assignHandlers from "./client"
import initModeration from "./mod"
import { persistMessages } from "./options"
import { watchThread } from './page/thread_watcher';

// Load all stateful modules in dependency order
async function start() {
	extractConfigs()

	await open()
	if (page.thread) {
		await loadFromDB(page.thread)
	}

	// Check for legacy options and remap
	const oldNowPlaying = localStorage.getItem("nowPlaying")
	switch (oldNowPlaying) {
	case "false":
	case "none":
		localStorage.setItem("nowPlaying", "" + (0 ^ 0));
		break;
	case "true":
	case "r/a/dio":
		localStorage.setItem("nowPlaying", "" + (0 ^ 1));
		break;
	case "eden":
		localStorage.setItem("nowPlaying", "" + (0 ^ 2));
		break;
	case "both":
		localStorage.setItem("nowPlaying", "" + (0 ^ 1 ^ 2));
		break;
	}

	initOptions()

	if (page.thread) {
		renderThread()

		// Open a cross-thread quoting reply
		connSM.once(connState.synced, () => {
			// Must run after other state handlers
			setTimeout(() => {
				const data = localStorage.getItem("openQuote");
				if (!data) {
					return;
				}
				const i = data.indexOf(":"),
					id = parseInt(data.slice(0, i)),
					sel = data.slice(i + 1);
				localStorage.removeItem("openQuote");
				if (posts.get(id)) {
					postSM.feed(postEvent.open);
					(trigger("getPostModel") as FormModel)
						.addReference(id, sel);
					requestAnimationFrame(scrollToBottom);
				}
			}, 100);
		})

		persistMessages()
		connect()
		checkBottom()
		assignHandlers()

		// Add a stored thread OP, made by the client to "mine" and set thread
		// as watched
		const addMine = getCookie("addMine");
		if (addMine) {
			const id = parseInt(addMine);
			storeMine(id, id);
			storeSeenPost(id, id);
			watchThread(id, 1, 0, page.board, thread.subject);
			deleteCookie("addMine");
		}
	} else {
		await renderBoard()
	}

	initPage()
	initPosts()
	initUI()
	initModeration()
}

start().catch(err => {
	alert(err.message)
	throw err
})
