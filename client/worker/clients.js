/*
 Functions for sending and receiving messages from clients
*/

import {randomID, WeakSetMap, message} from '../common'
import {extend} from 'underscore'

export const byID = {}
export const byPage = new WeakSetMap()

// Establish a private communication channel and SW-side instance of tab. Only
// these assingnment requests are sent through the global SW channel.
self.onmessage = msg =>
	new Client(msg)

// Handles interctions with the client browser tabs. In an MVC sense, these
// contain only the View part.
class Client {
	constructor(msg) {
		this.port = msg.ports[0]
		this.port.onmessage = this.receive.bind(this)
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
	setState(url) {
		this.unsetState()
		const state = this.parseURL(url)
		extend(this, state)
		const {board, thread} = state
		byPage.add(thread ? thread : board, this)
	}

	// Remove client from client byPage map
	unsetState() {
		byPage.remove(this.board, this)
		byPage.remove(this.thread, this)
	}

	// Parse the client URL into a board name, thread number and last to display
	// post number setting
	parseURL(href) {
		const state = {
			board: href.match(/\/([a-zA-Z0-9]+?)\//)[1],
			thread: href.match(/\/(\d+)(:?#\d+)?(?:[\?&]\w+=\w+)*$/),
			lastN: href.match(/[\?&]last=(\d+)/)
		}
		for (let key of ['thread', 'lastN']) {
			const val = state[key]
			state[key] = val ? parseInt(val[1]) : 0
		}
		return state
	}

	// Remove the client's instance
	remove() {
		this.unsetState()
		this.port.close()
		delete byID[this.id]
	}

	// Receive and parse messages from the browser tab
	receive(msg) {
		switch (msg.data) {
		case message.disconnect:
			this.remove()
			break
		default:
			throw new Error('Unknown client message type: ' + msg.data)
		}
	}
}
