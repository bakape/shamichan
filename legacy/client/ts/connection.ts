/*
 Handles Websocket connectivity
*/

// Message types of the WebSocket communication protocol
export const message: {[type: string]: number} = {

}

// Tokenization map of sync statuses
const enum syncStatus  {disconnected, syncing, synced}

let connection: WebSocket,
	reconnTimer: any

const path = (location.protocol === 'https' ? 'wss' : 'ws')
	+ location.host + '/socket'

function connect() {
	connection = new WebSocket(path)
	connection.binaryType = "arraybuffer"
	connection.onmessage = onMessage
	connection.onopen = onOpen
	connection.onclose = onClose
}

connect()

function onMessage(msg: MessageEvent) {
	console.log(msg)
}

function onOpen() {
	if (reconnTimer) {
		clearInterval(reconnTimer)
		reconnTimer = null
	}
}

function onClose() {
	if (!reconnTimer) {
		reconnTimer = setInterval(connect, 5000)
		sendAll(tabMessage.syncStatus, [syncStatus.disconnected])
	}
}
