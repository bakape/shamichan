// Core websocket message handlers

import { handlers, message, connSM, connEvent } from './connection'
import { posts, page, mine } from './state'
import { Post, PostView } from './posts'
import { PostData } from "./common"
import { postAdded, OverlayNotification } from "./ui"
import { incrementPostCount } from "./page"
import { posterName } from "./options"

// Run a function on a model, if it exists
function handle(id: number, fn: (m: Post) => void) {
	const model = posts.get(id)
	if (model) {
		fn(model)
	}
}

// Insert a post into the models and DOM
export function insertPost(data: PostData) {
	// R/a/dio song name override
	if (posterName()) {
		data.name = posterName()
	}

	const model = new Post(data)
	model.op = page.thread
	model.board = page.board
	if (mine.has(model.id)) {
		model.seenOnce = true
	}
	posts.add(model)
	const view = new PostView(model, null)

	model.propagateLinks()

	// Find last allocated post and insert after it
	const last = document
		.getElementById("thread-container")
		.lastElementChild
	if (last.id === "p0") {
		last.before(view.el)
	} else {
		last.after(view.el)
	}

	postAdded(model)
	incrementPostCount(true, "image" in data)

	// Show new post separator
	if (document.hidden) {
		let hr = document.getElementById("new-post-hr")
		if (!hr) {
			hr = document.createElement("hr")
			hr.id = "new-post-hr"
			view.el.before(hr)
		}
	}
}

export default () => {
	handlers[message.invalid] = (msg: string) => {

		// TODO: More user-friendly critical error reporting

		alert(msg)
		connSM.feed(connEvent.error)
		throw msg
	}

	handlers[message.insertPost] = insertPost

	handlers[message.spoiler] = (id: number) =>
		handle(id, m =>
			m.spoilerImage())

	handlers[message.deletePost] = (id: number) =>
		handle(id, m =>
			m.setDeleted())

	handlers[message.deleteImage] = (id: number) =>
		handle(id, m =>
			m.removeImage())

	handlers[message.banned] = (id: number) =>
		handle(id, m =>
			m.setBanned())

	handlers[message.redirect] = (board: string) =>
		location.href = `/${board}/`

	handlers[message.notification] = (text: string) =>
		new OverlayNotification(text)
}
