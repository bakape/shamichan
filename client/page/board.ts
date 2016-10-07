import { random, escape, on } from '../util'
import { navigation, ui, time } from '../lang'
import { boardConfig, page } from '../state'
import { ThreadData } from '../posts/models'
import { renderThumbnail } from '../posts/render/image'
import options from '../options'
import { write, $threads, importTemplate } from '../render'
import { setTitle } from "../tab"
import { formatText, renderNotice } from "./common"
import { renderTime } from "../posts/render/posts"
import { fetchBoard } from "../json"

type SortFunction = (a: ThreadData, b: ThreadData) => number

// Thread sort functions
const sorts: { [name: string]: SortFunction } = {
	bump: (a, b) =>
		b.bumpTime - a.bumpTime,
	lastReply: (a, b) =>
		b.replyTime - a.replyTime,
	creation: (a, b) =>
		b.time - a.time,
	replyCount: (a, b) =>
		b.postCtr - a.postCtr,
	fileCount: (a, b) =>
		b.imageCtr - a.imageCtr,
}

// Cached data of the current board's threads
let data: ThreadData[],
	// Unix time of last board page render. Used for automatic refreshes.
	lastRender: number,
	// Progress counter of the current board. Used for skipping useless renders.
	progressCounter: number

// Format a board name and title into cannonical board header format
export function formatHeader(name: string, title: string): string {
	return `/${name}/ - ${escape(title)}`
}

// Cache the curent board contents and render the thread
export default function cachetAndRender(threads: ThreadData[], ctr: number) {
	data = threads
	progressCounter = ctr
	lastRender = Math.floor(Date.now() / 1000)
	render(threads)
}

// Render a board page's HTML
function render(threads: ThreadData[]) {
	const frag = importTemplate("board")

	// Apply board title to tab and header
	const title = formatHeader(page.board, boardConfig.title)
	setTitle(title)
	frag.querySelector(".page-title").innerHTML = title

	const {banners} = boardConfig
	if (banners.length) {
		const banner = frag.querySelector(".image-banner") as HTMLElement
		banner.hidden = false
		banner
			.firstElementChild
			.setAttribute("src", `/assets/banners/${random(banners)}`)
	}

	// Render rules container aside
	const {rules} = boardConfig
	if (!rules || page.board === "all") {
		(frag.querySelector("#rules") as HTMLElement).style.display = "none"
	} else {
		const $rc = frag.querySelector("#rules-container")
		if (!rules) {
			$rc.append("God's in his heaven. All is right with the world.")
		} else {
			$rc.append(formatText(rules))
		}
	}
	(frag.querySelector("select[name=sortMode]") as HTMLSelectElement)
		.value = localStorage.getItem("catalogSort") || "bump"

	renderRefreshButton(frag.querySelector("#refresh"))

	renderNotice(frag)
	frag.querySelector("#catalog").append(renderThreads("", threads))

	write(() => {
		$threads.innerHTML = ""
		$threads.append(frag)
	})
}

// Sort, filter and render all threads on a board
function renderThreads(
	filter: string, threads: ThreadData[],
): DocumentFragment {
	if (filter) {
		const r = new RegExp(filter, "i")
		threads = threads.filter(({board, subject}) =>
			r.test(`/${board}/`) || r.test(subject))
	}

	threads = threads.sort(sorts[localStorage.getItem("catalogSort") || "bump"])

	const frag = document.createDocumentFragment(),
		threadEls: DocumentFragment[] = new Array(threads.length)
	for (let i = 0; i < threads.length; i++) {
		threadEls[i] = renderThread(threads[i])
	}
	frag.append(...threadEls)
	return frag
}

// Render a single thread for the thread catalog
function renderThread(thread: ThreadData): DocumentFragment {
	const frag = importTemplate("catalog-thread"),
		href = `../${thread.board}/${thread.id}`,
		lastN = options.lastN.toString()

	frag.firstElementChild.id = "p" + thread.id

	if (thread.image) {
		thread.image.large = true // Display larger thumbnails
		if (!options.hideThumbs && !options.workModeToggle) {
			const fig = frag.querySelector("figure")
			fig.hidden = false
			renderThumbnail(fig.querySelector("a"), thread.image, href)
		}
	}

	const $links = frag.querySelector(".thread-links")
	const $board = $links.querySelector(".board") as HTMLElement
	$board.hidden = false
	$board.textContent = `/${thread.board}/`
	$links
		.querySelector(".counters")
		.textContent = `${thread.postCtr}/${thread.imageCtr}`
	const $lastN = $links.querySelector("a.history")
	$lastN.setAttribute("href", `${href}?last=${lastN}`)
	$lastN.textContent = `${navigation.last} ${lastN}`

	frag.querySelector("h3").innerHTML = `「${escape(thread.subject)}」`

	return frag
}

// Render the board refresh button text
function renderRefreshButton(el: Element) {
	renderTime(el, lastRender, true)
	if (el.textContent === time.justNow) {
		el.textContent = ui.refresh
	}
}

// Toggle the [Rules] cotainer expansion or contraction
function toggleRules(e: MouseEvent) {
	const $el = e.target as HTMLElement,
		$aside = $el.closest("aside")
	if ($aside.classList.contains("expanded")) {
		write(() => {
			$aside.classList.remove("expanded")
			$el.textContent = ui.rules
		})
	} else {
		write(() => {
			$aside.classList.add("expanded")
			$el.textContent = ui.close
		})
	}
}

// Persist thread sort order mode to localStorage and rerender threads
function onSortChange(e: Event) {
	localStorage.setItem("catalogSort", (e.target as HTMLInputElement).value)
	const filter =
		($threads.querySelector("input[name=search]") as HTMLInputElement)
			.value
	writeThreads(renderThreads(filter, data))
}

function writeThreads(threads: DocumentFragment) {
	const cat = $threads.querySelector("#catalog")
	write(() => {
		cat.innerHTML = ""
		cat.append(threads)
	})
}

// Refilter and rerender threads on seach input change
function onSearchChange(e: Event) {
	const threads = renderThreads((e.target as HTMLInputElement).value, data)
	writeThreads(threads)
}

// Fetch and rerender board contents
async function refreshBoard() {
	const {ctr, threads} = await fetchBoard(page.board)
	cachetAndRender(threads, ctr)
}

// Update refresh timer or refresh board, if document hidden, each minute
setInterval(() => {
	if (page.thread) {
		return
	}
	if (document.hidden) {
		refreshBoard()
	} else {
		write(() =>
			renderRefreshButton($threads.querySelector("#refresh")))
	}
}, 60000)

on($threads, "click", toggleRules, {
	passive: true,
	selector: "#rules a",
})

on($threads, "change", onSortChange, {
	passive: true,
	selector: "select[name=sortMode]",
})

on($threads, "input", onSearchChange, {
	passive: true,
	selector: "input[name=search]",
})

on($threads, "click", refreshBoard, {
	passive: true,
	selector: "#refresh",
})
