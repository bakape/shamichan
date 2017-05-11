import { fetchBoard, fetchThread, extend } from "../util"
import { PageState, posts, read, page } from '../state'
import renderThread from './thread'
import { renderFresh as renderBoard } from './board'
import { setExpandAll } from "../posts"
import initNavigation from "./navigation"

export { extractConfigs, isBanned } from "./common"
export {
	incrementPostCount, default as renderThread, setThreadTitle
} from "./thread"
export { render as renderBoard } from "./board"

initNavigation()

// Load a page (either board or thread) and render it once the ready promise
// has been resolved
export async function loadPage(state: PageState, ready: Promise<void>) {
	const { board, thread, lastN, catalog, href } = state
	const res = thread
		? await fetchThread(board, thread, lastN)
		: await fetchBoard(board, catalog)
	const t = await res.text()
	await ready
	switch (res.status) {
		case 200:
			// Possibly was redirected
			if (thread && board === "all") {
				const redir = read(res.url)

				// Strip internal query parameter
				const u = new URL(redir.href, location.origin)
				let query = u.search,
					url = u.pathname
				if (query) {
					query = query
						.slice(1)
						.split("&")
						.filter(p =>
							p !== "minimal=true")
						.join("&")
					if (query) {
						url += "?" + query
					}
				}

				// Restore old hash
				const hash = new URL(href, location.origin).hash
				if (hash) {
					url += hash
				}

				redir.href = url

				page.replaceWith(redir)
				extend(state, redir)
			}
			break
		case 403:
			break
		default:
			throw t
	}

	posts.clear()
	setExpandAll(false)
	if (thread) {
		renderThread(t)
	} else {
		renderBoard(t)
	}
}
