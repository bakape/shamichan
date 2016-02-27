/*
 Handles Websocket connectivity
*/

import {sendAll} from './clients'
import {message as tabMessage, syncStatus} from '../common'

// Message types of the WebSocket communication protocol
export const message: {[type: string]: number} = {

}

export let connection: Connection

// Websocket connection handler
class Connection extends WebSocket {
	constructor() {
		super('/socket')
		this.onmessage = msg => this.receive(msg)
	}

	receive(msg: MessageEvent) {
		console.log(msg)
	}
}

let reconnTimer: number

export function connect(): void {
	connection = new Connection()
	connection.onopen = () => {
		if (reconnTimer) {
			clearInterval(reconnTimer)
			reconnTimer = null
		}
	}
	connection.onclose = () => {
		if (!reconnTimer) {
			reconnTimer = setInterval(connect, 5000)
			sendAll(tabMessage.syncStatus, [syncStatus.disconnected])
		}
	}
}
