// Core websocket message handlers

import {handlers, message, connSM, connEvent} from './connection'
import {posts, mine} from './state'
import {Post, PostLinks} from './posts/models'

// Message for splicing the contents of the current line
export type SpliceResponse = {
	id: number
	start: number
	len: number
	text: string
}

// Message sent to listening clients about a link or backlink insertion into
// a post
type LinkMessage = {
	id: number
	links: PostLinks
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
handlers[message.link] = ({id, links}: LinkMessage) =>
	handle(id, m =>
		m.insertLink(links))
handlers[message.backlink] = ({id, links}: LinkMessage) =>
	handle(id, m =>
		m.insertBacklink(links))

// Run a function on model, if it exists
function handle(id: number, fn: (m: Post) => void) {
	const model = posts.get(id)
	if (model) {
		fn(model)
	}
}
