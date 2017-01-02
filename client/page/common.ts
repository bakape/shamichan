import { fetchBoard, fetchThread } from "../fetch"
import { PageState, posts, setBoardConfig } from '../state'
import renderThread from './thread'
import { renderFresh as renderBoard } from './board'
import { setExpandAll } from "../posts/images"

// Load a page (either board or thread) and render it once the ready promise
// has been resolved
export default async function (
	{board, thread, lastN}: PageState,
	ready: Promise<void>
) {
	const res = thread
		? await fetchThread(board, thread, lastN)
		: await fetchBoard(board)
	const t = await res.text()
	switch (res.status) {
		case 200:
		case 403:
			break
		default:
			throw t
	}

	await ready

	posts.clear()
	setExpandAll(false)
	if (thread) {
		renderThread(t)
	} else {
		renderBoard(t)
	}
}

// Find board configurations in the HTML and apply them
export function extractConfigs() {
	const conf = document.getElementById("board-configs").textContent
	setBoardConfig(JSON.parse(conf))
}

// Check if the rendered page is a ban page
export function isBanned(): boolean {
	return !!document.querySelector(".ban-page")
}
