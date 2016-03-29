/*
 Handles Websocket connectivity
*/

import FSM from './fsm'
import {debug} from './state'
import {sync as lang} from './lang'

// Message types of the WebSocket communication protocol
export const message: {[type: string]: number} = {

}

export type MessageHandler = (data: ArrayBuffer) => void

// Websocket message handlers. Each handler responds to its distinct message
// type.
export const handlers: {[type: number]: MessageHandler} = {}

// Websocket connection and syncronisation with server states
const enum syncStatus {disconnected, connecting, syncing, synced, desynced}

// States of the connection finite state machine
export const enum connState {
	loading, connecting, syncing, synced, reconnecting, dropped, desynced
}

// Events passable to the connection FSM
export const enum connEvent {
	start, open, close, retry, error
}

// Finite state machine for managing websocket connectivity
export const connSM = new FSM<connState, connEvent>(connState.loading)

let socket: WebSocket,
	attempts: number,
	attemptTimer: number

// Send a message to the server
export function send(msg: ArrayBuffer) {
	if (connSM.state !== connState.synced
		&& connSM.state !== connState.syncing
	) {
		return
	}
	if (socket.readyState !== 1) {
		console.warn("Attempting to send while socket closed")
		return
	}
	if (debug) {
		console.log('<', msg)
	}
	socket.send(msg)
}

// Routes messages from the server to the respective handler
function onMessage({data}: MessageEvent) {
	if (debug) {
		console.log('>', data)
	}
	const handler = handlers[data[0]]
	if (handler) {
		handler(data)
	}
}

const syncEl = document.getElementById('sync')

// Render connction status indicator
function renderStatus(status: syncStatus) {
	syncEl.textContent = lang[status]
}

connSM.act([connState.loading], connEvent.start, connState.connecting, () => {
	renderStatus(syncStatus.connecting)
	attempts = 0
	connect()
})

const path = (location.protocol === 'https' ? 'wss' : 'ws')
	+ `://${location.host}/socket`

function connect() {
	nullSocket()
	if (window.location.protocol == 'file:') {
		console.error("Page downloaded locally. Refusing to sync.")
		return
	}
	socket = new WebSocket(path)
	socket.binaryType = "arraybuffer"
	socket.onopen = connSM.feeder(connEvent.open)
	socket.onclose = connSM.feeder(connEvent.close)
	socket.onmessage = onMessage
	socket.onerror = onError
	if (debug) {
		(window as any).socket = socket
	}
}

function onError(err: Event) {
	console.error(err)
}

// Strip all handlers and remove references from Websocket instance
function nullSocket() {
	if (socket) {
		socket.onclose
			= socket.onmessage
			= socket.onopen
			= socket.onclose
			= socket.onerror
			= socket
			= null
	}
}

connSM.act(
	[connState.connecting, connState.reconnecting],
	connEvent.open,
	connState.syncing,
	() => {
		renderStatus(syncStatus.connecting)
		attemptTimer = setTimeout(() => resetAttempts(), 10000)
	}
)

// Reset the reconnection attempt counter and timers
function resetAttempts() {
	if (attemptTimer) {
		clearTimeout(attemptTimer)
		attemptTimer = 0
	}
	attempts = 0
}

connSM.wildAct(connEvent.close, connState.dropped, err => {
	nullSocket()
	if (debug) {
		console.error(err)
	}
	if (attemptTimer) {
		clearTimeout(attemptTimer)
		attemptTimer = 0
	}
	renderStatus(syncStatus.disconnected)

	// Wait maxes out at ~1min
	const wait = 500 * Math.pow(
		1.5,
		Math.min(Math.floor(++attempts / 2), 12)
	)
	setTimeout(connSM.feeder(connEvent.retry), wait)
})

connSM.act([connState.dropped], connEvent.retry, connState.reconnecting, () => {
	connect()

	// Don't show this immediately so we don't thrash on network loss
	setTimeout(() => {
		if (connSM.state === connState.reconnecting) {
			renderStatus(syncStatus.connecting)
		}
	}, 100)
})

export function start() {
	connSM.feed(connEvent.start)
}

// Work arround browser slowing down/suspending tabs and keep the FSM up to date
// with the actual status.
function onWindowFocus() {
	if (connSM.state !== connState.desynced && navigator.onLine) {
		connSM.feed(connEvent.retry)
	}
}

document.addEventListener('visibilitychange', event => {
	if (!(event.target as Document).hidden) {
		onWindowFocus()
	}
})

window.addEventListener('online', () => {
	resetAttempts()
	connSM.feed(connEvent.retry)
})
window.addEventListener('offline', connSM.feeder(connEvent.close))
