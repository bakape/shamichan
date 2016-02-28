/*
 Handles Websocket connectivity
*/

import {sendAll} from './clients'
import {message as tabMessage, syncStatus} from '../common'

// Message types of the WebSocket communication protocol
export const message: {[type: string]: number} = {

}

let connection: Connection,
	reconnTimer: any

// Websocket connection handler
class Connection extends WebSocket {
	constructor() {
		let path = location.protocol === 'https' ? 'wss' : 'ws'
		path += location.host + '/socket'
		super(path)
		this.binaryType = "arraybuffer"
		this.onmessage = msg => this.onMessage(msg)
		this.onopen = () => this.onOpen()
		this.onclose = () => this.onClose()
	}

	onMessage(msg: MessageEvent) {
		console.log(msg)
	}

	onOpen() {
		if (reconnTimer) {
			clearInterval(reconnTimer)
			reconnTimer = null
		}
	}

	onClose() {
		if (!reconnTimer) {
			reconnTimer = setInterval(connect, 5000)
			sendAll(tabMessage.syncStatus, [syncStatus.disconnected])
		}
	}
}

export function connect(): void {
	connection = new Connection()
}
