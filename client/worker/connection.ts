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

	private onMessage(msg: MessageEvent) {
		console.log(msg)
	}

	private onOpen() {
		if (reconnTimer) {
			clearInterval(reconnTimer)
			reconnTimer = null
		}
	}

	private onClose() {
		if (!reconnTimer) {
			reconnTimer = setInterval(connect, 5000)
			sendAll(tabMessage.syncStatus, [syncStatus.disconnected])
		}
	}
}

function connect() {
	connection = new Connection()
}

connect()
