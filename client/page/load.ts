import {fetchJSON, fetchBoarConfigs} from '../util'
import {PageState, boardConfig, posts, setSyncCounter} from '../state'
import renderThread from './thread'
import renderBoard from './board'
import {ThreadData} from '../posts/models'

// Data of a single board retrieved from the server through `/json/:board`
type BoardData = {
	ctr: number
	threads: ThreadData[]
}

// Load a page (either board or thread) and render it once the ready promise
// has been resolved
export default async function (
	{board, thread}: PageState,
	ready: Promise<void>
) {
	const conf = fetchBoarConfigs(board)
	let data: BoardData|ThreadData

	if (thread) {
		data = await fetchJSON(`/json/${board}/${thread}`) as ThreadData
	} else {
		data = await fetchJSON(`/json/${board}/`) as BoardData
	}

	await ready
	boardConfig.replaceWith(await conf)
	setSyncCounter((data as ThreadData).logCtr || (data as BoardData).ctr)
	posts.clear()

	if (thread) {
		renderThread(data as ThreadData)
	} else {
		renderBoard((data as BoardData).threads)
	}
}
