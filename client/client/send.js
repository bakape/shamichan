/*
 Communication with the ServiceWorker backend
*/

import {message} from '../common'

const chan = new MessageChannel()

// Send a message to the ServiceWorker thorough the private channel
export function send(msg, transfers) {
	if (!transfers) {
		chan.port1.postMessage(msg)
	} else {
		chan.port1.postMessage(msg, transfers)
	}
}

// Establish a private message channel with the ServiceWorker
function connectToSW() {
	const cont = navigator.serviceWorker.controller
	if (!cont) {
		setTimeout(connectToSW, 10)
		return
	}
	cont.postMessage(location.href, [chan.port2])
}

connectToSW()

window.addEventListener("beforeunload", () =>
	send(message.disconnect))
