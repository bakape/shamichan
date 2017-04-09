import { FSM, trigger } from "../util"
import { debug, page } from "../state"
import { message, handlers } from "./messages"
import { renderStatus } from "./ui"
import { postSM, postEvent, postState, identity, FormModel } from "../posts"

// Websocket connection and synchronization with server states
export const enum syncStatus {
	disconnected, connecting, syncing, synced, desynced,
}

let socket: WebSocket,
	syncCounter: number, // Tracks thread update progress
	attempts: number,
	attemptTimer: number

// States of the connection finite state machine
export const enum connState {
	loading, connecting, syncing, synced, reconnecting, dropped, desynced
}

// Events passable to the connection FSM
export const enum connEvent {
	start, open, close, retry, error, sync,
}

// Finite state machine for managing websocket connectivity
export const connSM = new FSM<connState, connEvent>(connState.loading)

const path = (location.protocol === 'https:' ? 'wss' : 'ws')
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
	socket.onmessage = ({ data }) =>
		onMessage(data, false)
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

// Send a message to the server. If msg is null, it is omitted from sent
// websocket message.
export function send(type: message, msg: any) {
	if (socket.readyState !== 1) {
		console.warn("Attempting to send while socket closed")
		return
	}

	let str = leftPad(type)
	if (msg !== null) {
		str += JSON.stringify(msg)
	}

	if (debug) {
		console.log('<', str)
	}
	socket.send(str)
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
function onMessage(data: string, extracted: boolean) {
	// First two characters of a message define its type
	const type = parseInt(data.slice(0, 2))

	if (debug) {
		console.log(extracted ? "\t>" : ">", data)
	}

	// Split several concatenated messages
	if (type === message.concat) {
		for (let msg of data.slice(2).split('\u0000')) {
			onMessage(msg, true)
		}
		return
	}

	// Message types bellow thirty alter the thread state
	if (type < 30) {
		syncCounter++
	}

	const handler = handlers[type]
	if (handler) {
		handler(JSON.parse(data.slice(2)))
	}
}

// Update the thread synchronization progress counter
export function setSyncCounter(c: number) {
	syncCounter = c
}

function prepareToSync(): connState {
	renderStatus(syncStatus.connecting)
	synchronise()
	attemptTimer = setTimeout(resetAttempts, 10000) as any
	return connState.syncing
}

// Send a requests to the server to synchronise to the current page and
// subscribe to the appropriate event feeds
export function synchronise() {
	send(message.synchronise, {
		board: page.board,
		thread: page.thread,
	})

	// Reclaim a post lost after disconnecting, going on standby, resuming
	// browser tab, etc.
	if (page.thread && postSM.state === postState.halted) {
		// No older than 28 minutes
		const m = trigger("getPostModel") as FormModel
		if (m.time > (Date.now() / 1000 - 28 * 60)) {
			send(message.reclaim, {
				id: m.id,
				password: identity.postPassword,
			})
		} else {
			postSM.feed(postEvent.abandon)
		}
	}
}

function clearModuleState() {
	nullSocket()
	if (attemptTimer) {
		clearTimeout(attemptTimer)
		attemptTimer = 0
	}
}

// Work around browser slowing down/suspending tabs and keep the FSM up to date
// with the actual status.
function onWindowFocus() {
	if (!navigator.onLine) {
		return
	}
	switch (connSM.state) {
		// Ensure still connected, in case the computer went to sleep or
		// hibernate or the mobile browser tab was suspended.
		case connState.synced:
			send(message.NOOP, null)
			break
		case connState.desynced:
			break
		default:
			connSM.feed(connEvent.retry)
	}
}

// Reset the reconnection attempt counter and timers
function resetAttempts() {
	if (attemptTimer) {
		clearTimeout(attemptTimer)
		attemptTimer = 0
	}
	attempts = 0
}

export function start() {
	connSM.feed(connEvent.start)
}

// If thread data is too old because of disconnect, computer suspension or
// resuming old tabs, refetch and/or sync differences.
async function fetchBacklog(start: number, end: number) {
	const res = await fetch("/json/log", {
		method: "POST",
		body: JSON.stringify({
			id: page.thread,
			start, end,
		})
	})
	switch (res.status) {
		case 200:
			// Text body will be empty, if there are no messages
			const data = await res.text()
			if (data) {
				onMessage(data, false)
			}
			break
		default:
			throw await res.text()
	}
}

connSM.act(connState.loading, connEvent.start, () => {
	renderStatus(syncStatus.connecting)
	attempts = 0
	connect()
	return connState.connecting
})

for (let state of [connState.connecting, connState.reconnecting]) {
	connSM.act(state, connEvent.open, prepareToSync)
}

// Synchronise to the server and start receiving updates on the appropriate
// channel. If there are any missed messages, fetch them.
handlers[message.synchronise] = async (ctr: number) => {
	if (page.thread && ctr !== syncCounter) {
		await fetchBacklog(syncCounter, ctr).catch(alert)
	}
	connSM.feed(connEvent.sync)
}

// Handle response to a open post reclaim request
handlers[message.reclaim] = (code: number) => {
	switch (code) {
		case 0:
			postSM.feed(postEvent.reclaim)
			break
		case 1:
			postSM.feed(postEvent.abandon)
			break
	}
}

connSM.act(connState.syncing, connEvent.sync, () => {
	renderStatus(syncStatus.synced)
	return connState.synced
})

connSM.wildAct(connEvent.close, event => {
	clearModuleState()
	if (debug) {
		console.error(event)
	}
	renderStatus(syncStatus.disconnected)

	// Wait maxes out at ~1min
	const wait = 500 * Math.pow(1.5, Math.min(Math.floor(++attempts / 2), 12))
	setTimeout(connSM.feeder(connEvent.retry), wait)

	return connState.dropped
})

connSM.act(connState.dropped, connEvent.retry, () => {
	if (!navigator.onLine) {
		return connState.dropped
	}

	connect()

	// Don't show this immediately so we don't thrash on network loss
	setTimeout(() =>
		connSM.state === connState.reconnecting
		&& renderStatus(syncStatus.connecting)
		, 100)
	return connState.reconnecting
})

// Invalid message or some other critical error
connSM.wildAct(connEvent.error, () => {
	renderStatus(syncStatus.desynced)
	clearModuleState()
	return connState.desynced
})

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
