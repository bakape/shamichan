import {random, escape} from '../util'
import {navigation} from '../lang'
import {boardConfig, page} from '../state'
import {ThreadData} from '../posts/models'
import {renderThumbnail} from '../posts/render/image'
import options from '../options'
import {write, $threads, importTemplate} from '../render'
import {setTitle} from "../tab"

// Format a board name and title into cannonical board header format
export function formatHeader(name: string, title: string): string {
	return escape(`/${name}/ - ${title}`)
}

// Render a board page's HTML
export default function (threads: ThreadData[]) {

	// TODO: Cutomisable sorting order

	threads = threads.sort((a, b) =>
		b.bumpTime - a.bumpTime)

	const frag = importTemplate("board")

	// Apply board title to tab and header
	const title = formatHeader(page.board, boardConfig.title)
	setTitle(title)
	frag
		.querySelector(".page-title")
		.innerHTML = formatHeader(page.board, title)

	const {banners} = boardConfig
	if (banners.length) {
		const banner = frag.querySelector(".image-banner") as HTMLElement
		banner.hidden = false
		banner
			.firstElementChild
			.setAttribute("src", `/assets/banners/${random(banners)}`)
	}

	const threadEls: DocumentFragment[] = []
	for (let i = 0; i < threads.length; i++) {
		threadEls[i] = renderThread(threads[i])
	}
	frag.querySelector("#catalog").append(...threadEls)

	write(() => {
		$threads.innerHTML = ""
		$threads.append(frag)
	})
}

// Render a single thread for the thread catalog
function renderThread(thread: ThreadData): DocumentFragment {
	const frag = importTemplate("catalog-thread"),
		href = `../${thread.board}/${thread.id}`,
		lastN = options.lastN.toString()

	frag.firstElementChild.id = "p" + thread.id

	if (thread.image) {
		thread.image.large = true // Display larger thumbnails
		if (!options.hideThumbs) {
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
	$lastN.setAttribute("href", `${href}?lastN=${lastN}`)
	$lastN.textContent = `${navigation.last} ${lastN}`

	frag.querySelector("h3").innerHTML = `「${escape(thread.subject)}」`

	return frag
}
