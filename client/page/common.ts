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
	const [html, err] = thread
		? await fetchThread(board, thread, lastN)
		: await fetchBoard(board)
	if (err) {
		throw err
	}

	await ready

	posts.clear()
	setExpandAll(false)
	if (thread) {
		renderThread(html)
	} else {
		renderBoard(html)
	}
}

// Find board configurations in the HTML and apply them
export function extractConfigs() {
	const conf = document.getElementById("board-configs").textContent
	setBoardConfig(JSON.parse(conf))
}
