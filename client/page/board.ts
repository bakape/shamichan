import { escape, on, fetchBoard } from '../util'
import lang from '../lang'
import { page } from '../state'
import options from '../options'
import { renderTime } from "../posts"
import { setTitle } from "../ui"
import { extractConfigs, isBanned } from "."
import { setPostCount } from "./thread"
import { setSyncCounter } from "../connection"

type SortFunction = (a: HTMLElement, b: HTMLElement) => number

// Thread sort functions
const sorts: { [name: string]: SortFunction } = {
	bump: subtract("bump-time"),
	lastReply: subtract("reply-time"),
	creation: subtract("time"),
	replyCount: subtract("post-ctr"),
	fileCount: subtract("image-ctr"),
}
const threads = document.getElementById("threads")

// Unix time of last board page render. Used for automatic refreshes.
let lastFetchTime = Date.now() / 1000

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

// Render a board fresh board page
export function renderFresh(html: string) {
	setSyncCounter(0)
	lastFetchTime = Math.floor(Date.now() / 1000)
	threads.innerHTML = html
	if (isBanned()) {
		return
	}
	extractConfigs()
	render()
}

// Apply client-side modifications to a board page's HTML
export function render() {
	setPostCount(0, 0)

	// Apply board title to tab
	setTitle(threads.querySelector("#page-title").textContent)

	// Add extra localizations
	for (let el of threads.querySelectorAll(".counters")) {
		el.setAttribute("title", lang.ui["postsImages"])
	}
	for (let el of threads.querySelectorAll(".lastN-link")) {
		el.textContent = `${lang.ui["last"]} 100`
	}

	(threads.querySelector("select[name=sortMode]") as HTMLSelectElement)
		.value = localStorage.getItem("catalogSort") || "bump"
	renderRefreshButton(threads.querySelector("#refresh > a"))
	sortThreads(true)
}

// Sort all threads on a board
export function sortThreads(initial: boolean) {
	let catalog = document.getElementById("catalog"),
		threads = Array.from(catalog.querySelectorAll("article"))

	if (options.hideThumbs || options.workModeToggle) {
		for (let el of catalog.querySelectorAll("img.expanded")) {
			el.style.display = "none"
		}
	}

	const sortMode = localStorage.getItem("catalogSort") || "bump"
	// Already sorted as needed
	if (initial && sortMode === "bump") {
		return
	}

	for (let el of threads) {
		el.remove()
	}
	threads = threads.sort(sorts[sortMode])
	catalog.append(...threads)
}

// Render the board refresh button text
function renderRefreshButton(el: Element) {
	renderTime(el, lastFetchTime, true)
	if (el.textContent === lang.posts["justNow"]) {
		el.textContent = lang.ui["refresh"]
	}
}

// Persist thread sort order mode to localStorage and rerender threads
function onSortChange(e: Event) {
	localStorage.setItem("catalogSort", (e.target as HTMLInputElement).value)
	sortThreads(false)
}

function onSearchChange(e: Event) {
	const filter = (e.target as HTMLInputElement).value
	filterThreads(filter)
}

// Filter against board and subject and toggle thread visibility
function filterThreads(filter: string) {
	const r = new RegExp(filter, "i"),
		catalog = document.getElementById("catalog")

	for (let el of catalog.querySelectorAll("article")) {
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
	const res = await fetchBoard(page.board),
		t = await res.text()
	switch (res.status) {
		case 200:
		case 403:
			renderFresh(t)
			break
		default:
			throw t
	}
}

// Update refresh timer or refresh board, if document hidden, each minute
// TODO: Replace with SSE
setInterval(() => {
	if (page.thread || isBanned()) {
		return
	}
	if (document.hidden) {
		refreshBoard()
	} else {
		renderRefreshButton(threads.querySelector("#refresh > a"))
	}
}, 600000)

on(threads, "input", onSortChange, {
	passive: true,
	selector: "select[name=sortMode]",
})
on(threads, "input", onSearchChange, {
	passive: true,
	selector: "input[name=search]",
})
on(threads, "click", refreshBoard, {
	passive: true,
	selector: "#refresh > a",
})
