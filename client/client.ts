// Core websocket message handlers

import {register, message, connSM, connEvent} from './connection'

register(message.invalid, (msg: string) => {
	// TODO: More user-frienly critical error reporting
	alert(msg)
	connSM.feed(connEvent.error)
})
