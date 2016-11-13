import { escape, on, makeFrag } from '../util'
import { ui, time } from '../lang'
import { page } from '../state'
import options from '../options'
import { write, threads } from '../render'
import { setTitle } from "../tab"
import { renderTime } from "../posts/render/posts"
import { fetchBoard } from "../fetch"

type SortFunction = (a: HTMLElement, b: HTMLElement) => number

// Thread sort functions
const sorts: { [name: string]: SortFunction } = {
	lastReply: subtract("replyTime"),
	creation: subtract("time"),
	replyCount: subtract("postCtr"),
	fileCount: subtract("imageCtr"),
}

// Unix time of last board page render. Used for automatic refreshes.
let lastFetch = Date.now() / 1000

// Sort threads by embedded data
function subtract(attr: string): (a: HTMLElement, b: HTMLElement) => number {
	attr = "data-" + attr
	return (a, b) =>
		parseInt(b.getAttribute(attr)) - parseInt(a.getAttribute(attr))
}

// Format a board name and title into canonical board header format
export function formatHeader(name: string, title: string): string {
	return `/${name}/ - ${escape(title)}`
}

// Render a board fresh board from string
export function renderFresh(html: string) {
	lastFetch = Math.floor(Date.now() / 1000)

	const frag = makeFrag(html)
	render(frag)
	threads.innerHTML = ""
	threads.append(frag)
}

// Apply client-side modifications to a board page's HTML
export function render(frag: NodeSelector) {

	// Apply board title to tab
	setTitle(frag.querySelector("#page-title").textContent)

	// Set sort mode <select> to correspond with setting
	let sortMode = localStorage.getItem("catalogSort")
	// "bump" is a legacy sort mode. Account for clients explicitly set to it.
	if (sortMode === "bump") {
		sortMode = ""
		localStorage.removeItem("catalogSort")
	}
	if (!sortMode) {
		sortMode = "lastReply"
	}
	(frag.querySelector("select[name=sortMode]") as HTMLSelectElement)
		.value = sortMode

	renderRefreshButton(frag.querySelector("#refresh"))
	sortThreads(frag.querySelector("#catalog"), true)
}

// Sort all threads on a board
export function sortThreads(frag: ParentNode, initial: boolean) {
	let threads = Array.from(frag.children)

	if (options.hideThumbs || options.workModeToggle) {
		for (let el of threads) {
			el.querySelector(".expanded").style.display = "none"
		}
	}

	let sortMode = localStorage.getItem("catalogSort")
	if (!sortMode || sortMode === "bump") {
		sortMode = "lastReply"
	}

	// Already sorted as needed
	if (initial && sortMode === "lastReply") {
		return
	}

	for (let el of threads) {
		el.remove()
	}
	threads = threads.sort(sorts[sortMode])
	frag.append(...threads)
}

// Render the board refresh button text
function renderRefreshButton(el: Element) {
	renderTime(el, lastFetch, true)
	if (el.textContent === time.justNow) {
		el.textContent = ui.refresh
	}
}

// Persist thread sort order mode to localStorage and rerender threads
function onSortChange(e: Event) {
	localStorage.setItem("catalogSort", (e.target as HTMLInputElement).value)
	sortThreads(document.getElementById("catalog"), false)
}

function onSearchChange(e: Event) {
	const filter = (e.target as HTMLInputElement).value
	filterThreads(filter, document.getElementById("catalog"))
}

// Filter against board and subject and toggle thread visibility
function filterThreads(filter: string, catalog: ParentNode) {
	const r = new RegExp(filter, "i")

	for (let el of Array.from(catalog.children) as HTMLElement[]) {
		let display = "none"

		const board = el.querySelector(".board")
		if (board && r.test(board.textContent)) {
			display = ""
		} else {
			const subject = el.querySelector("h3").textContent.slice(1, -1)
			if (r.test(subject)) {
				display = ""
			}
		}

		el.style.display = display
	}
}


// Fetch and rerender board contents
async function refreshBoard() {
	renderFresh(await fetchBoard(page.board))
}

// Update refresh timer or refresh board, if document hidden, each minute
// TODO: Replace with SSE
setInterval(() => {
	if (page.thread) {
		return
	}
	if (document.hidden) {
		refreshBoard()
	} else {
		write(() =>
			renderRefreshButton(threads.querySelector("#refresh")))
	}
}, 600000)

on(threads, "change", onSortChange, {
	passive: true,
	selector: "select[name=sortMode]",
})
on(threads, "input", onSearchChange, {
	passive: true,
	selector: "input[name=search]",
})
on(threads, "click", refreshBoard, {
	passive: true,
	selector: "#refresh",
})
