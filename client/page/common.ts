import {
	fetchJSON, fetchBoarConfigs, makeFrag, escape, hashString, on,
} from '../util'
import {PageState, boardConfig, posts, setSyncCounter, page} from '../state'
import renderThread from './thread'
import renderBoard from './board'
import {ThreadData} from '../posts/models'
import {scrollToAnchor} from "../scroll"
import {read} from "../render"
import {images, ui} from "../lang"
import {write, $threads} from "../render"

// Data of a single board retrieved from the server through `/json/:board`
type BoardData = {
	ctr: number
	threads: ThreadData[]
}

// Load a page (either board or thread) and render it once the ready promise
// has been resolved
export default async function (
	{board, thread, lastN}: PageState,
	ready: Promise<void>
) {
	const conf = fetchBoarConfigs(board)
	let data: BoardData|ThreadData

	if (thread) {
		let url = `/json/${board}/${thread}`
		if (lastN) {
			url += `?lastN=${lastN}`
		}
		data = await fetchJSON<ThreadData>(url)
	} else {
		data = await fetchBoard(board)
	}

	await ready
	boardConfig.replaceWith(await conf)
	setSyncCounter((data as ThreadData).logCtr || (data as BoardData).ctr || 0)
	posts.clear()

	if (thread) {
		renderThread(data as ThreadData)
	} else {
		renderBoard((data as BoardData).threads)
	}

	// Scroll to any selected anchor, after page renders
	read(() =>
		scrollToAnchor())
}

// Fetch JSON data of a board page
export async function fetchBoard(board: string): Promise<BoardData> {
	return await fetchJSON<BoardData>(`/json/${board}/`)
}

// Format a block of text received from an untrusted user
export function formatText(s: string): DocumentFragment {
	return makeFrag(escape(s).replace(/\n/g, "<br>"))
}

// Render board-specific notices from the board owners
export function renderNotice(frag: NodeSelector) {
	const {notice} = boardConfig,
		$el = frag.querySelector("#notice") as HTMLElement

	if (!notice) {
		$el.style.display = "none"
		return
	}

	$el.lastElementChild.append(formatText(notice))
	const hash = hashString(notice).toString(),
		$a = $el.querySelector("a")
	if (hash === localStorage.getItem(noticeKey())) {
		$a.textContent = ui.showNotice
	} else {
		$el.classList.add("expanded")
		$a.textContent = images.hide
	}
}

function noticeKey(): string {
	return "notice:" + page.board
}

// Toggle display of the board notice and persist its hash, if already seen
function toggleNotice(e: MouseEvent) {
	const $el = e.target as HTMLElement,
		$div = $el.closest("div")
	if ($div.classList.contains("expanded")) {
		// Remember that the user has seen and hidden this notice
		const hash = hashString(boardConfig.notice).toString()
		localStorage.setItem(noticeKey(), hash)

		write(() => {
			$div.classList.remove("expanded")
			$el.textContent = ui.showNotice
		})
	} else {
		write(() => {
			$div.classList.add("expanded")
			$el.textContent = images.hide
		})
	}
}

on($threads, "click", toggleNotice, {
	passive: true,
	selector: "#notice a",
})
