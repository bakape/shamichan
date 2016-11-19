import { fetchBoard, fetchThread } from "../fetch"
import { PageState, posts } from '../state'
import renderThread from './thread'
import { renderFresh as renderBoard } from './board'
import { ThreadData } from '../posts/models'

// Load a page (either board or thread) and render it once the ready promise
// has been resolved
export default async function (
	{board, thread, lastN}: PageState,
	ready: Promise<void>
) {
	const [data, err] = thread
		? await fetchThread(board, thread, lastN)
		: await fetchBoard(board)
	if (err) {
		throw err
	}

	await ready

	posts.clear()

	if (thread) {
		renderThread(data as ThreadData)
	} else {
		renderBoard(data as string)
	}
}
