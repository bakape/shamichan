import { fetchBoard, fetchThread, extend } from "../util"
import { PageState, posts, setBoardConfig, read } from '../state'
import renderThread from './thread'
import { renderFresh as renderBoard } from './board'
import { setExpandAll } from "../posts"
import initNavigation from "./navigation"

export {
	incrementPostCount, default as renderThread, setThreadTitle
} from "./thread"
export { render as renderBoard } from "./board"

initNavigation()

// Load a page (either board or thread) and render it once the ready promise
// has been resolved
export async function loadPage(state: PageState, ready: Promise<void>) {
	const {board, thread, lastN} = state
	const res = thread
		? await fetchThread(board, thread, lastN)
		: await fetchBoard(board)
	const t = await res.text()
	switch (res.status) {
		case 200:
			// Was redirected
			if (thread && board === "cross") {
				const redir = read(res.url)

				// Strip internal query parameter
				let [url, query] = redir.href.split("?")
				if (query) {
					query = query
						.split("&")
						.filter(p =>
							p !== "noIndex=true")
						.join("&")
					if (query) {
						url += "?" + query
					}
				}
				const [, hash] = state.href.split("#")
				if (hash) {
					url += "#" + hash
				}
				redir.href = url

				extend(state, redir)
			}
			break
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
