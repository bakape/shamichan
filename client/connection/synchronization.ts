import { handlers, message } from "./messages"
import { connSM, connEvent, send } from "./state"
import { postSM, postEvent, postState, identity, FormModel } from "../posts"
import { page, posts } from "../state"
import { trigger } from "../util"
import { PostData } from "../common"
import { insertPost } from "../client"

// Passed from the server to allow the client to synchronise state, before
// consuming any incoming update messages.
type SyncData = {
	recent: number[] // Posts created within the last 15 minutes
	open: { [id: number]: OpenPost } // Posts currently open
}

// State of an open post
type OpenPost = {
	hasImage: boolean
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
		// No older than 28 minutes
		const m = trigger("getPostModel") as FormModel
		if (m.time > (Date.now() / 1000 - 28 * 60)) {
			send(message.reclaim, {
				id: m.id,
				password: identity.postPassword,
			})
		} else {
			postSM.feed(postEvent.abandon)
		}
	}
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
handlers[message.synchronise] = async ({ open, recent }: SyncData) => {
	const proms: Promise<void>[] = []

	for (let id in open) {
		proms.push(syncOpenPost(parseInt(id), open[id]))
	}

	for (let id of recent) {
		// Missing posts, that are open, will be fetched by the loop above
		if (!posts.get(id) && !open[id]) {
			proms.push(fetchMissingPost(id))
		}
	}

	try {
		await Promise.all(proms)
	} catch (e) {
		return alert(e)
	}
	connSM.feed(connEvent.sync)
}

// Sync open posts to the state they are in on the server's update feed
// dispatcher
async function syncOpenPost(id: number, { hasImage, body }: OpenPost) {
	let model = posts.get(id)
	if (!model) {
		await fetchMissingPost(id)
		model = posts.get(id)
	}
	if (hasImage && !model.image) {
		model.image = (await fetchPost(id)).image
		model.view.renderImage(false)
	}
	model.body = body
	model.view.reparseBody()
}

// Fetch a post not present on the client and render it
async function fetchMissingPost(id: number) {
	insertPost(await fetchPost(id))
	posts.get(id).view.reposition()
}

async function fetchPost(id: number): Promise<PostData> {
	const r = await fetch(`/json/post/${id}`)
	if (r.status !== 200) {
		throw await r.text()
	}
	return r.json()
}
