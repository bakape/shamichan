import { handlers, message } from "./messages"
import { connSM, connEvent, send } from "./state"
import { page, posts, displayLoading } from "../state"
import { insertPost } from "../client"
import { uncachedGET } from "../util"
import { PostData } from "../common"

// Passed from the server to allow the client to synchronise state, before
// consuming any incoming update messages
type SyncData = {
	replies: number[]
	banned: number[]
	deleted: number[]
	spoilered: number[]
	deletedImages: number[]
}

// Send a requests to the server to synchronise to the current page and
// subscribe to the appropriate event feeds
export function synchronise() {
	send(message.synchronise, {
		board: page.board,
		thread: page.thread,
	})
}

// Fetch a post not present on the client and render it
async function fetchMissingPost(id: number) {
	insertPost(await fetchPost(id))
	posts.get(id).view.reposition()
}

async function fetchPost(id: number): Promise<PostData> {
	const r = await uncachedGET(`/json/post/${id}`)
	if (r.status !== 200) {
		throw await r.text()
	}
	return r.json()
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

	const { replies, banned, deleted, spoilered, deletedImages, } = data,
		proms: Promise<void>[] = []

	for (let id of replies) {
		if (!posts.get(id) && id > minID) {
			proms.push(fetchMissingPost(id))
		}
	}
	for (let id of banned) {
		const p = posts.get(id)
		if (p && !p.banned) {
			p.setBanned()
		}
	}
	for (let id of deleted) {
		const p = posts.get(id)
		if (p && !p.deleted) {
			p.setDeleted()
		}
	}
	for (let id of spoilered) {
		const p = posts.get(id)
		if (p && p.image && !p.image.spoiler) {
			p.image.spoiler = true
			p.view.renderImage(false)
		}
	}
	for (let id of deletedImages) {
		const p = posts.get(id)
		if (p && p.image) {
			p.removeImage()
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
