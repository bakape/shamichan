// Tab title and favicon rendering

import {connSM, connState} from "./connection"
import {write} from "./render"

const $title = document.head.querySelector("title"),
	$favicon = document.head.querySelector("#favicon"),
	urlBase = "/assets/favicons/"

let title: string,
	discoFavicon: string

// Set the text part of a page title
export function setTitle(t: string) {
	title = t
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

	apply("", urlBase + "default.ico")

	// TODO: Unread and qouted indications

}

// Write tab title and favicon to DOM
function apply(prefix: string, favicon: string) {
	write(() =>
		($title.innerHTML = prefix + title,
		$favicon.setAttribute("href", favicon)))
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
