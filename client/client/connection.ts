/*
 Communication with the ServiceWorker backend and websocket connecttion status
 user notification.
*/

import {message} from '../common'

export const handlers: {(msg: MessageEvent): void}[] = []
const chan = new MessageChannel()

// Handle messages from the SW
chan.port1.onmessage = msg => {
	const fn = handlers[msg.data]
	if (!fn) {
		throw new Error('Unknown client message type: ' + msg.data)
	}
	fn(msg)
}

// Send a message to the ServiceWorker thorough the private channel
export function send(msg: any, transfers?: any[]) {
	if (!transfers) {
		chan.port1.postMessage(msg)
	} else {
		chan.port1.postMessage(msg, transfers)
	}
}

// Establish a private message channel with the ServiceWorker
export function connect() {
	const cont = navigator.serviceWorker.controller
	if (!cont) {
		setTimeout(connect, 10)
		return
	}
	cont.postMessage(location.href, [chan.port2])
}

// Disconnect from SW, when navigating away from the page
window.addEventListener("beforeunload", () =>
	send(message.disconnect))

const syncEl = document.query('#sync')

handlers[message.syncStatus] = msg => {
	console.log(msg)
}
