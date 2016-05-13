/*
 Handles Websocket connectivity
*/

import FSM from './fsm'
import {debug, syncCounter, page, clientID} from './state'
import {sync as lang} from './lang'
import {write} from './render'

// A reqeust message to synchronise or resynchronise (after a connection loss)
// to the server
type SyncRequest = {
	board: string
	thread: number
	ctr: number
	id?: string
}

// Message types of the WebSocket communication protocol
export const enum message {
	invalid,

	// 1 - 29 modify post model state
	insertThread,
	insertPost,

	// >= 30 are miscelenious and do not write to post models
	synchronise = 30,
	resynchronise,
	switchSync,
}

export type MessageHandler = (msg: {}) => void

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
export function send(type: message, msg: {}) {
	if (connSM.state !== connState.synced
		&& connSM.state !== connState.syncing
		&& type !== message.synchronise
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
	socket.send(leftPad(type) + JSON.stringify(msg))
}

// Ensure message type is always a 2 characters long string
function leftPad(type: message): string {
	let str = type.toString()
	if (str.length === 1) {
		str = '0' + str
	}
	return str
}

// Routes messages from the server to the respective handler
function onMessage({data}: MessageEvent) {
	if (debug) {
		console.log('>', data)
	}

	// First two charecters of a message define its type
	const handler = handlers[parseInt(data.slice(0, 2))]
	if (handler) {
		handler(JSON.parse(data.slice(2)))
	}
}

const syncEl = document.getElementById('sync')

// Render connction status indicator
function renderStatus(status: syncStatus) {
	write(() => syncEl.textContent = lang[status])
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
	socket.onopen = connSM.feeder(connEvent.open)
	socket.onclose = connSM.feeder(connEvent.close)
	socket.onerror = connSM.feeder(connEvent.close)
	socket.onmessage = onMessage
	if (debug) {
		(window as any).socket = socket
	}
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

		// Send a requests to the server to syschronise to the current page and
		// subscribe to the apropriate event feeds.
		const msg: SyncRequest = {
			board: page.get('board'),
			thread: page.get('thread'),
			ctr: syncCounter,
		}
		let type = message.synchronise

		// If clientID is set, then this attempt to synchronise comes after a
		// connection loss. Attempt to recover lost server-side state.
		if (clientID) {
			msg.id = clientID
			type = message.resynchronise
		}

		send(type, msg)

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

connSM.wildAct(connEvent.close, connState.dropped, event => {
	nullSocket()
	if (debug) {
		console.error(event)
	}
	if (attemptTimer) {
		clearTimeout(attemptTimer)
		attemptTimer = 0
	}
	if (event.code !== 1000) {
		alert(`Websocket error: ${event.reason}`)
		renderStatus(syncStatus.desynced)
		return
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
