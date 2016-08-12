// Core websocket message handlers

import {handlers, message, connSM, connEvent} from './connection'
import {posts} from './state'
import {Post} from './posts/models'

handlers[message.invalid] = (msg: string) => {

	// TODO: More user-frienly critical error reporting

	alert(msg)
	connSM.feed(connEvent.error)
}

handlers[message.append] = ([id, char]: number[]) =>
	handle(id, m =>
		m.append(char))

// Run a function on model, if it exists
function handle(id: number, fn: (m: Post) => void) {
	const model = posts.get(id)
	if (model) {
		fn(model)
	}
}
