// Core websocket message handlers

import {handlers, message, connSM, connEvent} from './connection'
import {posts} from './state'
import {Post} from './posts/models'

// Message for splicing the contents of the current line
export type SpliceResponse = {
	id: number
	start: number
	len: number
	text: string
}

handlers[message.invalid] = (msg: string) => {

	// TODO: More user-frienly critical error reporting

	alert(msg)
	connSM.feed(connEvent.error)
}

handlers[message.append] = ([id, char]: number[]) =>
	handle(id, m =>
		m.append(char))
handlers[message.backspace] = (id: number) =>
	handle(id, m =>
		m.backspace())
handlers[message.splice] = (msg: SpliceResponse) =>
	handle(msg.id, m =>
		m.splice(msg))

// Run a function on model, if it exists
function handle(id: number, fn: (m: Post) => void) {
	const model = posts.get(id)
	if (model) {
		fn(model)
	}
}
