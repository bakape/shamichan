// Core websocket message handlers

import {handlers, message, connSM, connEvent} from './connection'
import {posts} from './state'
import {Post, PostLinks, Command, PostData, ImageData} from './posts/models'
import PostView from "./posts/view"
import {$threadContainer} from "./page/thread"
import {write} from "./render"

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

// Meesage to inject a new command result into a model
interface CommandMessage extends Command {
	id: number
}

// Message for inserting images into an open post
interface ImageMessage extends ImageData {
	id: number
}

// Run a function on a model, if it exists
function handle(id: number, fn: (m: Post) => void) {
	const model = posts.get(id)
	if (model) {
		fn(model)
	}
}

handlers[message.invalid] = (msg: string) => {

	// TODO: More user-frienly critical error reporting

	alert(msg)
	connSM.feed(connEvent.error)
}

handlers[message.insertPost] = (data: PostData) => {
	// If the post is already in the global collection, it was just created by
	// this client
	const mine = posts.get(data.id)
	if (mine) {
		if (data.image) {
			mine.insertImage(data.image)
		}
		return
	}

	const model = new Post(data)
	posts.add(model)
	const view = new PostView(model)
	write(() =>
		$threadContainer.append(view.el))

	// TODO: Hooks for triggering desktop notifications

}

handlers[message.insertImage] = (msg: ImageMessage) =>
	handle(msg.id, m =>
		(delete msg.id,
		m.insertImage(msg)))

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

handlers[message.command] = (msg: CommandMessage) =>
	handle(msg.id, m =>
		(delete msg.id,
		m.insertCommand(msg)))

handlers[message.closePost] = (id: number) =>
	handle(id, m =>
		m.closePost())
