// Tab title and favicon rendering

import { connSM, connState } from "./connection"
import { deferInit } from "./defer"

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
	if (document.hidden) {
		unseenPosts++
		resolve()
	}
}

// Add enseen reply indicator to tab header
export function repliedToMe() {
	if (document.hidden) {
		unseenReplies = true
		resolve()
	}
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
	if (document.hidden) {
		if (unseenPosts) {
			prefix = `(${unseenPosts}) `
			icon = "unread"
		}
		if (unseenReplies) {
			prefix = ">> " + prefix
			icon = "reply"
		}
	}
	apply(prefix, `${urlBase}${icon}.ico`)
}

// Write tab title and favicon to DOM. If we use requestAnimationFrame here,
// this will never render on a hidden document.
function apply(prefix: string, favicon: string) {
	$title.innerHTML = prefix + title
	$favicon.setAttribute("href", favicon)
}

// Account for immeadiate recconection and only render favicon, if not
// reconnected in 5 seconds
function delayedDiscoRender() {
	setTimeout(() => {
		switch (connSM.state) {
			case connState.dropped:
			case connState.desynced:
				resolve()
		}
	}, 5000)
}

deferInit(() => {
	// Needs to be available with no connectivity, so we download and cache it
	fetch(urlBase + "disconnected.ico")
		.then(res =>
			res.blob())
		.then(blob =>
			discoFavicon = URL.createObjectURL(blob))

	// Connection change listeners
	connSM.on(connState.synced, resolve)
	for (let state of [connState.dropped, connState.desynced]) {
		connSM.on(state, delayedDiscoRender)
	}

	// Reset title on tab focus
	document.addEventListener('visibilitychange', () => {
		if (!document.hidden) {
			unseenPosts = 0
			unseenReplies = false
			resolve()
		}
	})
})
