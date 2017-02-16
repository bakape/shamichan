// Core websocket message handlers

import { handlers, message, connSM, connEvent } from './connection'
import { posts, page } from './state'
import { Post, FormModel, PostView, postEvent, postSM } from './posts'
import { PostLink, Command, PostData, ImageData } from "./common"
import { postAdded, navigate } from "./ui"
import { incrementPostCount } from "./page"

// Message for splicing the contents of the current line
export type SpliceResponse = {
	id: number
	start: number
	len: number
	text: string
}

type CloseMessage = {
	id: number
	links: PostLink[] | null
	commands: Command[] | null
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

// Insert a post into the models and DOM
function insertPost(data: PostData) {
	const existing = posts.get(data.id)
	if (existing) {
		if (existing instanceof FormModel && !existing.isAllocated) {
			existing.onAllocation(data)
			incrementPostCount(true, "image" in data)
		}
		return
	}

	const model = new Post(data)
	model.op = page.thread
	posts.add(model)
	const view = new PostView(model, null)

	// Find last allocated post and insert after it
	const last = document
		.getElementById("thread-container")
		.lastElementChild
	if (last.id === "p0") {
		last.before(view.el)
	} else {
		last.after(view.el)
	}

	postAdded()
	incrementPostCount(true, "image" in data)
}

export default () => {
	handlers[message.invalid] = (msg: string) => {

		// TODO: More user-friendly critical error reporting

		alert(msg)
		connSM.feed(connEvent.error)
	}

	handlers[message.insertPost] = insertPost

	handlers[message.insertImage] = (msg: ImageMessage) =>
		handle(msg.id, m => {
			delete msg.id
			if (!("image" in m)) {
				incrementPostCount(false, true)
			}
			m.insertImage(msg)
		})

	handlers[message.spoiler] = (id: number) =>
		handle(id, m =>
			m.spoilerImage())

	handlers[message.append] = ([id, char]: [number, number]) =>
		handle(id, m =>
			m.append(char))

	handlers[message.backspace] = (id: number) =>
		handle(id, m =>
			m.backspace())

	handlers[message.splice] = (msg: SpliceResponse) =>
		handle(msg.id, m =>
			m.splice(msg))

	handlers[message.backlink] = ([id, target, targetOP]: number[]) =>
		handle(id, m =>
			m.insertBacklink(target, targetOP))

	handlers[message.closePost] = ({id, links, commands}: CloseMessage) =>
		handle(id, m => {
			if (links) {
				m.links = links
				m.checkRepliedToMe()
			}
			if (commands) {
				m.commands = commands
			}
			m.closePost()
		})

	handlers[message.deletePost] = (id: number) =>
		handle(id, m =>
			m.remove())

	handlers[message.banned] = (id: number) =>
		handle(id, m =>
			m.setBanned())

	handlers[message.redirect] = (board: string) => {
		postSM.feed(postEvent.reset)
		navigate(`/${board}/`, null, true)
	}
}
