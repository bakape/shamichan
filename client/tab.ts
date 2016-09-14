// Tab title and favicon rendering

import {connSM, connState} from "./connection"

const $title = document.head.querySelector("title"),
	$favicon = document.head.querySelector("#favicon"),
	urlBase = "/assets/favicons/"

let title: string,
	unseenPosts = 0,
	unseenReplies = false,
	discoFavicon: string

// Set the text part of a page title
export function setTitle(t: string) {
	title = t
	resolve()
}

// Incerement unseen post number, if tab is hidden
export function postAdded() {
	unseenPosts++
	resolve()
}

// Add enseen reply indicator to tab header
export function repliedToMe() {
	unseenReplies = true
	resolve()
}

// Resolve tab title and favicon
function resolve() {
	switch (connSM.state) {
	case connState.desynced:
		return apply("--- ", urlBase + "error.ico")
	case connState.dropped:
		return apply("--- ", discoFavicon)
	}

	let prefix = "",
		icon = "default"
	if (unseenPosts) {
		prefix = `(${unseenPosts}) `
		icon = "unread"
	}
	if (unseenReplies) {
		prefix = ">> " + prefix
		icon = "reply"
	}
	apply(prefix, `${urlBase}${icon}.ico`)
}

// Write tab title and favicon to DOM. If we use requestAnimationFrame here,
// this will never render on a hidden document.
function apply(prefix: string, favicon: string) {
	$title.innerHTML = prefix + title
	$favicon.setAttribute("href", favicon)
}

// Needs to be available with no connectivity, so we download and cache it
fetch(urlBase + "disconnected.ico")
	.then(res =>
		res.blob())
	.then(blob =>
		discoFavicon = URL.createObjectURL(blob))

// Connection change listeners
for (let state of [connState.dropped, connState.desynced, connState.synced]) {
	connSM.on(state, resolve)
}

// Reset title on tab focus
document.addEventListener('visibilitychange', () => {
	if (!document.hidden) {
		unseenPosts = 0
		unseenReplies = false
		resolve()
	}
})
