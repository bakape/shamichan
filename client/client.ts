// Core websocket message handlers

import {message, handlers} from './connection'

type SyncResponse = {
	// ID of the client on the server. This should be stored and used to
	// reconnect on connection loss.
	id: string

	// New syncronisation counter to use
	sync: number

	// Backlog of messages the client was behind on prior to synchronisation
	// in historical order
	backlog: string[]
}

// Syncronise to the server and start receiving updates on the apropriate
// channel
handlers[message.synchronise] = (msg: SyncResponse) => {

}
