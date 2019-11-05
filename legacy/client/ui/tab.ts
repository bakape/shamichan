// Tab title and favicon rendering

import { connSM, connState } from "../connection"
import { Post } from "../posts"
import { posts } from "../state"

const titleEl = document.head.querySelector("title"),
	title = titleEl.textContent,
	faviconEl = document.getElementById("favicon"),
	urlBase = "/assets/favicons/",
	queue: Post[] = []

// All possible favicon states
const enum states { default, disconnected, error, unread, replied }

// Last state rendered as a favicon. Used to reduce DOM & tab header writes
const lastRendered = {
	state: states.default,
	unseenPosts: 0,
}

let unseenPosts = 0,
	unseenReplies = false,
	discoFavicon: string

// Update unseen post count based on post visibility and scroll position
export function postAdded(post: Post) {
	// async batch processing since visibility calculations force a layout
	if (queue.length == 0) {
		// can't use RAF since it's disabled in background tabs
		setTimeout(processQueue, 16)
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
			return apply("--- ", states.error)
		case connState.dropped:
			return apply("--- ", states.disconnected)
	}

	let prefix = "",
		state = states.default
	if (unseenPosts) {
		state = states.unread
		prefix = `(${unseenPosts}) `
	}
	if (unseenReplies) {
		state = states.replied
		prefix = ">> " + prefix
	}
	apply(prefix, state)
}

let recalcPending = false

function recalc() {
	recalcPending = false
	unseenPosts = 0
	unseenReplies = false
	for (let post of posts) {
		if (post.seen()) {
			continue;
		}
		unseenPosts++
		if (post.isReply()) {
			unseenReplies = true
		}
	}
	resolve()
}

// Write tab title and favicon to DOM
function apply(prefix: string, state: states) {
	// Same data - skip write to DOM
	if (lastRendered.state === state
		&& lastRendered.unseenPosts === unseenPosts
	) {
		return
	}

	lastRendered.unseenPosts = unseenPosts
	lastRendered.state = state

	titleEl.innerHTML = prefix + title
	let url: string
	switch (state) {
		case states.default:
			url = urlBase + "default.ico"
			break
		case states.disconnected:
			url = discoFavicon
			break
		case states.error:
			url = urlBase + "error.ico"
			break
		case states.replied:
			url = urlBase + "reply.ico"
			break
		case states.unread:
			url = urlBase + "unread.ico"
			break
	}
	faviconEl.setAttribute("href", url)
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

	document.addEventListener(
		"scroll",
		() => {
			if (recalcPending || document.hidden) {
				return
			}
			recalcPending = true
			setTimeout(recalc, 200)
		},
		{ passive: true },
	)
}
