// Core websocket message handlers

import { handlers, message, connSM, connEvent } from './connection'
import { posts } from './state'
import { Post, PostLinks, Command, PostData, ImageData } from './posts/models'
import FormModel from "./posts/posting/model"
import PostView from "./posts/view"
import { threadContainer } from "./page/thread"
import { write } from "./render"
import { postAdded } from "./tab"
import { deferInit } from "./defer"

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

// Message to inject a new command result into a model
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

// Insert a post into the models and DOM. The passed post may already exist and
// be rendered, in which case it is a possibly updated version, that syncs the
// client's state to the update stream. In that case the client must rerender
// posts or deduplicate appropriately.
export function insertPost(data: PostData) {
	// It is possible to receive insertion updates for posts that are not
	// currently displayed, because of the Last N setting. Skip them.
	if (data.id < posts.lowestID) {
		return
	}

	const existing = posts.get(data.id)
	if (existing) {
		if (existing instanceof FormModel) {
			if (!existing.isAllocated) {
				existing.onAllocation(data)
			}
		} else {
			existing.extend(data)
		}
		return
	}

	const model = new Post(data)
	posts.add(model)
	const view = new PostView(model, null)

	// Find last allocated post and insert after it
	write(() => {
		const last = threadContainer.lastElementChild
		if (last.id === "p0") {
			last.before(view.el)
		} else {
			last.after(view.el)
		}
	})

	postAdded()
	if (model.links) {
		model.checkRepliedToMe(model.links)
	}
}

deferInit(() => {
	handlers[message.invalid] = (msg: string) => {

		// TODO: More user-friendly critical error reporting

		alert(msg)
		connSM.feed(connEvent.error)
	}

	handlers[message.insertPost] = insertPost

	handlers[message.insertImage] = (msg: ImageMessage) =>
		handle(msg.id, m => {
			delete msg.id
			m.insertImage(msg)
		})

	handlers[message.spoiler] = (id: number) =>
		handle(id, m =>
			m.spoilerImage())

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
		handle(msg.id, m => {
			delete msg.id
			m.insertCommand(msg)
		})

	handlers[message.closePost] = (id: number) =>
		handle(id, m =>
			m.closePost())
})
