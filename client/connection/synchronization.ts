import { handlers, message } from "./messages"
import { connSM, connEvent, send } from "./state"
import { page, posts, displayLoading } from "../state"

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
}

// // Fetch a post not present on the client and render it
// async function fetchMissingPost(id: number) {
// 	insertPost(await fetchPost(id))
// 	posts.get(id).view.reposition()
// }

// async function fetchPost(id: number): Promise<PostData> {
// 	const r = await uncachedGET(`/json/post/${id}`)
// 	if (r.status !== 200) {
// 		throw await r.text()
// 	}
// 	return r.json()
// }

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

	// TODO
	// const { open, recent, banned, deleted, deletedImage } = data,
	// 	proms: Promise<void>[] = []

	// for (let post of posts) {
	// 	if (post.editing && !(post.id in open)) {
	// 		proms.push(fetchUnclosed(post))
	// 	}
	// }
	// for (let key in open) {
	// 	const id = parseInt(key)
	// 	if (id >= minID) {
	// 		proms.push(syncOpenPost(id, open[key]))
	// 	}
	// }
	// for (let id of recent) {
	// 	// Missing posts, that are open, will be fetched by the loop above
	// 	if (id >= minID && !posts.get(id) && !open[id]) {
	// 		proms.push(fetchMissingPost(id))
	// 	}
	// }
	// for (let id of banned) {
	// 	const post = posts.get(id)
	// 	if (post && !post.banned) {
	// 		post.setBanned()
	// 	}
	// }
	// for (let id of deleted) {
	// 	const post = posts.get(id)
	// 	if (post && !post.deleted) {
	// 		post.setDeleted()
	// 	}
	// }
	// for (let id of deletedImage) {
	// 	const post = posts.get(id)
	// 	if (post && post.image) {
	// 		post.removeImage()
	// 	}
	// }

	// if (proms.length) {
	// 	await Promise.all(proms).catch(e => {
	// 		alert(e)
	// 		throw e
	// 	})
	// }

	displayLoading(false)
	connSM.feed(connEvent.sync)
}
