import { on, fetchBoard } from '../util'
import lang from '../lang'
import { page, posts, loadFromDB, displayLoading, hidden } from '../state'
import options from '../options'
import { relativeTime, Post, findSyncwatches } from "../posts"
import {
	extractConfigs, extractPost, reparseOpenPosts, extractPageData, hidePosts,
} from "./common"
import { BoardData, ThreadData } from "../common"

type SortFunction = (a: Post, b: Post) => number

// Thread sort functions
const sorts: { [name: string]: SortFunction } = {
	bump: subtract("bump_time"),
	lastReply: subtract("update_time"),
	creation: subtract("time"),
	replyCount: subtract("post_count"),
	fileCount: subtract("image_count"),
}
const threadsEl = document.getElementById("threads")

// Thread currently loaded on the page
export const threads: { [id: number]: ThreadData } = {};

// Unix time of last board page render. Used for automatic refreshes.
let lastFetchTime = Date.now() / 1000

// Sort threads by embedded data
function subtract(attr: string): (a: Post, b: Post) => number {
	return (a, b) =>
		b[attr] - a[attr]
}

// Render a fresh board page
export function renderFresh(html: string) {
	lastFetchTime = Math.floor(Date.now() / 1000)
	threadsEl.innerHTML = html
	if (isBanned()) {
		return
	}
	extractConfigs()
	render()
}

// Check if the rendered page is a ban page
function isBanned(): boolean {
	return !!document.querySelector(".ban-page")
}

async function extractCatalogModels() {
	const data = extractPageData<BoardData>();
	await loadFromDB(...(data.threads.threads).map(t => t.id));
	
	for (let t of data.threads.threads) {
		threads[t.id] = t;
		extractPost(t, t.id, t.board, data.backlinks)
		if (hidden.has(t.id)) {
			document.getElementById(`p${t.id}`).classList.add("hidden")
		}
	}
}

async function extractThreads() {
	const data = extractPageData<BoardData>();
	await loadFromDB(...(data.threads.threads).map(t => t.id));

	for (let thread of data.threads.threads) {
		const { posts } = thread
		delete thread.posts
		threads[thread.id] = thread;
		extractPost(thread, thread.id, thread.board, data.backlinks)
		if (hidden.has(thread.id)) {
			document.querySelector(`section[data-id="${thread.id}"]`).classList.add("hidden")
			continue
		}
		for (let post of posts) {
			extractPost(post, thread.id, thread.board, data.backlinks)
		}
	}
	hidePosts()
	reparseOpenPosts()
}

// Apply client-side modifications to a board page's HTML
export async function render() {
	if (page.catalog) {
		await extractCatalogModels()
	} else {
		await extractThreads()
	}

	renderRefreshButton(threadsEl.querySelector("#refresh > a"))
	if (!page.catalog) {
		findSyncwatches(threadsEl)
	} else {
		(threadsEl.querySelector("select[name=sortMode]") as HTMLSelectElement)
			.value = localStorage.getItem("catalogSort") || "bump"
		sortThreads(true)
	}
	displayLoading(false)
}

// Sort all threads on a board
export function sortThreads(initial: boolean) {
	// Index pages are paginated, so it does not make a lot of sense to sort
	// them
	if (!page.catalog) {
		return
	}

	const [cont, threads] = getThreads()

	// Index board pages use the same localization functions as threads
	if (page.catalog && (options.hideThumbs || options.workModeToggle)) {
		for (let el of cont.querySelectorAll("img.catalog") as NodeListOf<HTMLElement>) {
			el.style.display = "none"
		}
	}

	const sortMode = localStorage.getItem("catalogSort") || "bump"
	// Already sorted as needed
	if (initial && sortMode === "bump") {
		return
	}

	// Sort threads by model properties
	const els: { [id: number]: HTMLElement } = {}
	cont.append(...threads
		.map(el => {
			const id = el.getAttribute("data-id")
			els[id] = el
			el.remove()
			return posts.get(parseInt(id))
		})
		.sort(sorts[sortMode])
		.map(({ id }) =>
			els[id])
	)
}

// Retrieves the thread container and the threads within depending on page type
function getThreads(): [HTMLElement, HTMLElement[]] {
	let contID: string,
		threadTag: string
	if (page.catalog) {
		contID = "catalog"
		threadTag = "article"
	} else {
		contID = "index-thread-container"
		threadTag = "section"
	}
	const cont = document.getElementById(contID)
	return [
		cont,
		Array.from(cont.querySelectorAll(threadTag)),
	]
}

// Render the board refresh button text
function renderRefreshButton(el: Element) {
	let text = relativeTime(lastFetchTime)
	if (text === lang.posts["justNow"]) {
		text = lang.ui["refresh"]
	}
	el.textContent = text
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
	const [, threads] = getThreads(),
		r = new RegExp(filter, "i"),
		matched = new Set<number>()
	for (let m of posts) {
		const match = (m.board && r.test(`/${m.board}/`))
			|| r.test(m.subject)
			|| r.test(m.body)
		if (match) {
			matched.add(m.op)
		}
	}

	for (let el of threads) {
		const id = parseInt(el.getAttribute("data-id"))
		el.style.display = matched.has(id) ? "" : "none"
	}
}

// Fetch and rerender board contents
async function refreshBoard() {
	const res = await fetchBoard(page.board, page.page, page.catalog),
		t = await res.text()
	switch (res.status) {
		case 200:
		case 403:
			posts.clear()
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
		renderRefreshButton(threadsEl.querySelector("#refresh > a"))
	}
}, 600000)

on(threadsEl, "input", onSortChange, {
	passive: true,
	selector: "select[name=sortMode]",
})
on(threadsEl, "input", onSearchChange, {
	passive: true,
	selector: "input[name=search]",
})
on(threadsEl, "click", refreshBoard, {
	passive: true,
	selector: "#refresh > a",
})
