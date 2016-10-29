// Handles Websocket connectivity and messaging

import FSM from './fsm'
import { debug, page, posts } from './state'
import { sync as lang } from './lang'
import { write } from './render'
import { authenticate } from './mod/login'
import { PostData } from "./posts/models"
import { insertPost } from "./client"
import { fetchThread } from "./json"
import identity from "./posts/posting/identity"
import { postSM, postEvent, postState, postModel } from "./posts/posting/main"

// A reqeust message to synchronise or resynchronise (after a connection loss)
// to the server
type SyncRequest = {
	board: string
	thread: number
	id?: string
}

// Message types of the WebSocket communication protocol
export const enum message {
	invalid,

	// 1 - 29 modify post model state
	insertThread,
	insertPost,
	append,
	backspace,
	splice,
	closePost,
	link,
	backlink,
	command,
	insertImage,
	spoiler,

	// >= 30 are miscelenious and do not write to post models
	synchronise = 30,
	reclaim,
	switchSync,

	// Account management
	register,
	login,
	authenticate,
	logout,
	logoutAll,
	changePassword,

	// Board and server administration
	configServer,
	createBoard,

	// Send new post ID to client
	postID,

	// Concatenation of multiple websocket messages to reduce transport overhead
	concat,
}

export type MessageHandler = (msg: {}) => void

// Websocket message handlers. Each handler responds to its distinct message
// type.
export const handlers: { [type: number]: MessageHandler } = {}

// Websocket connection and syncronisation with server states
const enum syncStatus { disconnected, connecting, syncing, synced, desynced }

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

let socket: WebSocket,
	attempts: number,
	attemptTimer: number,
	// Timestamp of the last post altering message received or the the initial
	// fetch of thread contents
	syncTimestamp: number

const syncEl = document.getElementById('sync')
const path =
	(location.protocol === 'https:' ? 'wss' : 'ws')
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
	socket.onmessage = ({data}) =>
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

// Render connction status indicator
function renderStatus(status: syncStatus) {
	write(() =>
		syncEl.textContent = lang[status])
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
	// First two charecters of a message define its type
	const type = parseInt(data.slice(0, 2))

	if (debug) {
		console.log(extracted ? ">>" : ">", data)
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
		updateSyncTimestamp()
	}

	const handler = handlers[type]
	if (handler) {
		handler(JSON.parse(data.slice(2)))
	}
}

// Update the thread synchronisation timestamp
export function updateSyncTimestamp() {
	syncTimestamp = Date.now()
}

function prepareToSync(): connState {
	renderStatus(syncStatus.connecting)
	synchronise(true)
	attemptTimer = setTimeout(() => resetAttempts(), 10000) as any
	return connState.syncing
}

// Send a requests to the server to syschronise to the current page and
// subscribe to the apropriate event feeds and optionally try to send a logged
// in user session authentication request.
export async function synchronise(auth: boolean) {
	const msg: SyncRequest = {
		board: page.board,
		thread: page.thread,
	}

	// If thread data is too old because of disconnect, computer suspention or
	// resuming old tabs, refetch and sync differences. The actual deadline
	// is 30 seconds, but a ten second buffer is probably sound.
	if (page.thread && Date.now() - syncTimestamp > 20000) {
		const {board, thread} = page,
			// Always fetch the full thread
			data = await fetchThread(board, thread, 0)
		insertPost(data)
		// ID of the first non-OP post that we have rendered, or OP, if none
		const firstID = Object.keys(posts.models).sort()[1] || page.thread
		for (let post of data.posts) {
			// Filter posts that we never retrived in lastN mode
			if (!posts.has(post.id) && post.id < firstID) {
				continue
			}
			insertPost(post)
		}
		delete data.posts
	}

	send(message.synchronise, msg)
	if (auth) {
		authenticate()
	}

	// Reclaim a post lost during after disconnecting, going on standby,
	// resuming browser tab, etc.
	if (
		page.thread
		&& postSM.state === postState.halted
		// No older than 28 minutes
		&& postModel.time < Date.now() / 1000 + 28 * 60
	) {
		send(message.reclaim, {
			id: postModel.id,
			password: identity.postPassword,
		})
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

function clearModuleState() {
	nullSocket()
	if (attemptTimer) {
		clearTimeout(attemptTimer)
		attemptTimer = 0
	}
}

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

connSM.act(connState.loading, connEvent.start, () => {
	renderStatus(syncStatus.connecting)
	attempts = 0
	connect()
	return connState.connecting
})

for (let state of [connState.connecting, connState.reconnecting]) {
	connSM.act(state, connEvent.open, prepareToSync)
}

// Syncronise to the server and start receiving updates on the apropriate
// channel. If there are any missed meessages, fetch them.
handlers[message.synchronise] = (backlog: { [id: number]: PostData }) => {
	if (page.thread) {
		for (let id in backlog) {
			insertPost(backlog[id])
		}
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
