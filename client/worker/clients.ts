/*
 Functions for sending and receiving messages from clients
*/

import {randomID, WeakSetMap, message} from '../common'
import {extend} from 'underscore'

export const byID: {[id: string]: Client} = {}
export const byPage = new WeakSetMap<number|string, Client>()
export const handlers: {(msg: MessageEvent, client: Client)}[] = []

// Establish a private communication channel and SW-side instance of tab. Only
// these assingnment requests are sent through the global SW channel.
self.onmessage = msg =>
	new Client(msg)

interface ClientState {
	board: string;
	thread: number;
	lastN: number;
}

// Handles interactions with the client browser tabs. In an MVC sense, these
// contain only the View part.
class Client {
	port: MessagePort;
	id: string;
	board: string;
	thread: number;
	lastN: number;

	constructor(msg: MessageEvent) {
		this.port = msg.ports[0]
		this.port.onmessage = msg => this.receive(msg)
		this.setState(msg.data)

		// Dedup client ID
		let id
		do {
			id = randomID(8)
		} while (id in byID)
		this.id = id
		byID[id] = this
	}

	// Replace previous client state with new one parsed from the supplied URL,
	// assing to the client lookup map
	setState(url: string) {
		this.unsetState()
		const state = this.parseURL(url)
		extend(this, state)
		const {board, thread} = state
		byPage.add(thread ? thread : board, this)
	}

	// Remove client from client byPage map
	unsetState(): void {
		byPage.remove(this.board, this)
		byPage.remove(this.thread, this)
	}

	// Parse the client URL into a board name, thread number and last to display
	// post number setting
	parseURL(href: string): ClientState {
		const board = href.match(/\/([a-zA-Z0-9]+?)\//)[1],
			thread = href.match(/\/(\d+)(:?#\d+)?(?:[\?&]\w+=\w+)*$/),
			lastN = href.match(/[\?&]last=(\d+)/)
		return {
			board,
			thread: thread ? parseInt(thread[1]) : 0,
			lastN: lastN ? parseInt(lastN[1]) : 0,
		}
	}

	// Remove the client's instance
	remove(): void {
		this.unsetState()
		this.port.close()
		delete byID[this.id]
	}

	// Receive and handle messages from the browser tab
	receive(msg: MessageEvent): void {
		const fn = handlers[msg.data]
		if (!fn) {
			throw new Error('Unknown client message type: ' + msg.data)
		}
		fn(msg, this)
	}

	// Send a message to the browser tab
	send(type: number, msg: any[]) {
		this.port.postMessage(type, [msg])
	}
}

handlers[message.disconnect] = (msg, client) =>
	client.remove()

// Send a message to all clients
export function sendAll(type: number, msg: any[]) {
	for (let id in byID) {
		byID[id].send(type, msg)
	}
}
