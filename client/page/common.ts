import { fetchBoard, fetchThread } from "../fetch"
import { PageState, posts, boardConfig } from '../state'
import renderThread from './thread'
import { renderFresh as renderBoard } from './board'
import { makeFrag } from "../util"
import { setTitle } from "../tab"
import { setExpandAll } from "../posts/images"
import { write } from "../render"

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
	const frag = makeFrag(html)
	extractConfigs(frag)
	// Apply board title to tab
	setTitle(frag.querySelector("#page-title").textContent)
	setExpandAll(false)

	if (thread) {
		renderThread(frag, true)
	} else {
		renderBoard(frag)
	}
}

// Find board configurations in the HTML and apply them
export function extractConfigs(ns: NodeSelector) {
	const conf = ns.querySelector("#board-configs").textContent
	boardConfig.replaceWith(JSON.parse(conf))
}

export function maybeWriteNow(writeNow: boolean, fn: () => void) {
    if (writeNow) {
        fn()
    } else {
        write(fn)
    }
}
