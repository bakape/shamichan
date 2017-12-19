import { handlers, message } from "./messages"
import { connSM, connEvent, send } from "./state"
import {
	postSM, postEvent, postState, identity, FormModel, Post
} from "../posts"
import { page, posts, displayLoading } from "../state"
import { trigger, uncachedGET, extend } from "../util"
import { PostData } from "../common"
import { insertPost } from "../client"

// Passed from the server to allow the client to synchronise state, before
// consuming any incoming update messages.
type SyncData = {
	recent: number[] // Posts created within the last 15 minutes
	open: { [id: number]: OpenPost } // Posts currently open
	deleted: number[] // Posts deleted
	deletedImage: number[] // Posts deleted in this thread
	banned: number[] // Posts banned in this thread
}

// State of an open post
type OpenPost = {
	hasImage: boolean
	spoilered: boolean
	body: string
}

// Send a requests to the server to synchronise to the current page and
// subscribe to the appropriate event feeds
export function synchronise() {
	send(message.synchronise, {
		board: page.board,
		thread: page.thread,
	})

	// Reclaim a post lost after disconnecting, going on standby, resuming
	// browser tab, etc.
	if (page.thread && postSM.state === postState.halted) {
		// No older than 15 minutes
		const m = trigger("getPostModel") as FormModel
		if (m.time > (Date.now() / 1000 - 15 * 60)) {
			send(message.reclaim, {
				id: m.id,
				password: identity.postPassword,
			})
		} else {
			postSM.feed(postEvent.abandon)
		}
	}
}

// Sync open posts to the state they are in on the server's update feed
// dispatcher
async function syncOpenPost(
	id: number,
	{ hasImage, body, spoilered }: OpenPost,
) {
	let model = posts.get(id)

	if (!model) {
		await fetchMissingPost(id)
		model = posts.get(id)
	} else if (model instanceof FormModel && model.editing) {
		// Don't rerender post form text
		model.inputBody = model.body = body
		model.view.onInput()
		return
	}

	if (hasImage && !model.image) {
		// Possible conflict due to deleted image
		if (model.image = (await fetchPost(id)).image) {
			model.view.renderImage(false)
		}
	}
	if (spoilered && !model.image.spoiler) {
		model.image.spoiler = true
		model.view.renderImage(false)
	}
	if (body) {
		model.body = body
	}
	model.view.reparseBody()
}

// Fetch a post not present on the client and render it
async function fetchMissingPost(id: number) {
	insertPost(await fetchPost(id))
	posts.get(id).view.reposition()
}

// Fetch a post that should be closed, but isn't
async function fetchUnclosed(post: Post) {
	extend(post, await fetchPost(post.id))
	post.propagateLinks()
	post.view.render()
}

async function fetchPost(id: number): Promise<PostData> {
	const r = await uncachedGET(`/json/post/${id}`)
	if (r.status !== 200) {
		throw await r.text()
	}
	return r.json()
}

// Handle response to a open post reclaim request
handlers[message.reclaim] = (code: number) => {
	switch (code) {
		case 0:
			postSM.feed(postEvent.reclaim)
			break
		case 1:
			postSM.feed(postEvent.abandon)
			break
	}
}

// Synchronise to the server and start receiving updates on the appropriate
// channel. If there are any missed messages, fetch them.
handlers[message.synchronise] = async (data: SyncData) => {
	if (!page.thread) {
		return
	}

	// Skip posts before the first post in a shortened thread
	let minID = 0
	if (page.lastN) {
		minID = Infinity
		for (let { id } of posts) {
			if (id < minID && id !== page.thread) {
				minID = id
			}
		}
		// No replies ;_;
		if (minID === Infinity) {
			minID = page.thread
		}
	}

	const { open, recent, banned, deleted, deletedImage } = data,
		proms: Promise<void>[] = []

	for (let post of posts) {
		if (post.editing && !(post.id in open)) {
			proms.push(fetchUnclosed(post))
		}
	}
	for (let key in open) {
		const id = parseInt(key)
		if (id >= minID) {
			proms.push(syncOpenPost(id, open[key]))
		}
	}
	for (let id of recent) {
		// Missing posts, that are open, will be fetched by the loop above
		if (id >= minID && !posts.get(id) && !open[id]) {
			proms.push(fetchMissingPost(id))
		}
	}
	for (let id of banned) {
		const post = posts.get(id)
		if (post && !post.banned) {
			post.setBanned()
		}
	}
	for (let id of deleted) {
		const post = posts.get(id)
		if (post && !post.deleted) {
			post.setDeleted()
		}
	}
	for (let id of deletedImage) {
		const post = posts.get(id)
		if (post && post.image) {
			post.removeImage()
		}
	}

	if (proms.length) {
		await Promise.all(proms).catch(e => {
			alert(e)
			throw e
		})
	}

	displayLoading(false)
	connSM.feed(connEvent.sync)
}
