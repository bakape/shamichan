interface Websocket {
	onmessage: WebsocketEventListener
	onopen: WebsocketEventListener
	onclose: WebsocketEventListener
	onerror: WebsocketEventListener
}

type WebsocketEventListener = (event: WebsocketEvent) => void

interface WebsocketEvent extends Event {
	data: any
	code: number
	reason: string
}
