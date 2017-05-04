// Tab title and favicon rendering

import { connSM, connState } from "../connection"
import { Post } from "../posts"
import { posts, page } from "../state"

const titleEl = document.head.querySelector("title"),
	faviconEl = document.getElementById("favicon"),
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

const queue : Post[] = [];

// Update unseen post count based on post visibility and scroll position
export function postAdded(post: Post) {
	// async batch processing since visibility calculations force a layout
	if(queue.length == 0) {
		requestAnimationFrame(processQueue)
	}

	queue.push(post)
}

function processQueue() {
	for (let post of queue) {
		if (!post.seen()) {
			unseenPosts++
		}
	}
	queue.length = 0
	resolve()
}

// Add unseen reply indicator to tab header
export function repliedToMe(post: Post) {
	if (!post.seen()) {
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

let recalcPending = false

function recalc() {
	recalcPending = false
	unseenPosts = 0
	unseenReplies = false
	for(let post of posts) {
		if(post.seen()) {
			continue;
		}
		unseenPosts++
		if(post.isReply()) {
			unseenReplies = true
		}
	}
	resolve()
}

// Write tab title and favicon to DOM. If we use requestAnimationFrame here,
// this will never render on a hidden document.
function apply(prefix: string, favicon: string) {
	titleEl.innerHTML = prefix + title
	faviconEl.setAttribute("href", favicon)
}

// Account for immediate reconnection and only render favicon, if not
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

export default () => {
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

	page.onChange("thread", () => {
		unseenPosts = 0
		unseenReplies = false
		resolve()
	})

	document.addEventListener("scroll", () => {
		if(recalcPending || document.hidden) {
			return
		}
		recalcPending = true
		setTimeout(recalc, 200)
	}, {passive: true})
}
