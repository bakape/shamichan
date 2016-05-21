// Core websocket message handlers

import {handlers, message, connSM, connEvent} from './connection'

handlers[message.invalid] = (msg: string) => {
	// TODO: More user-frienly critical error reporting
	alert(msg)
	connSM.feed(connEvent.error)
}
