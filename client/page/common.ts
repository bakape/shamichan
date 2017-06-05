import { setBoardConfig, hidden, mine, posts, page } from "../state"
import options from "../options"
import { PostData, ThreadData, fileTypes } from "../common"
import { Post, PostView } from "../posts"
import lang from "../lang"
import { postAdded } from "../ui/tab"
import { pluralize } from "../util"
import { posterName } from "../options"

// Post data of a particular page
export type PageData = {
	threads: ThreadData | ThreadData[],
	backlinks: { [id: number]: { [id: number]: number } },
}

const threads = document.getElementById("threads")

// Find board configurations in the HTML and apply them
export function extractConfigs() {
	setBoardConfig(extractJSON("board-configs"))
}

function extractJSON(id: string): any {
	const el = document.getElementById(id)
	if (!el) {
		return null
	}
	return JSON.parse(el.textContent)
}

// Extract pregenerated rendered post data from DOM
export function extractPageData(): PageData {
	return {
		threads: extractJSON("post-data"),
		backlinks: extractJSON("backlink-data"),
	}
}

// Check if the rendered page is a ban page
export function isBanned(): boolean {
	return !!document.querySelector(".ban-page")
}

// Extract post model and view from the HTML fragment and apply client-specific
// formatting. Returns whether the element was removed.
export function extractPost(
	post: PostData,
	op: number,
	board: string,
	replies: number[],
	backlinks: { [id: number]: { [id: number]: number } },
): boolean {
	const el = document.getElementById(`p${post.id}`)
	if (hidden.has(post.id)) {
		el.remove()
		return true
	}
	post.op = op
	post.board = board

	const model = new Post(post),
		view = new PostView(model, el)
	posts.add(model)

	if (page.catalog) {
		return false
	}
	model.backlinks = backlinks[post.id]

	// Apply client-specific formatting to a post rendered server-side

	// Render time-zone correction or relative time. Will do unneeded work,
	// if client is on UTC. Meh.
	view.renderTime()

	// Localize staff titles
	if (post.auth && options.lang != "en_GB") {
		view.renderName()
	}

	localizeLinks(model, replies)
	localizeBacklinks(model)
	postAdded(model)

	const { image } = model
	if (image) {
		const should = options.hideThumbs
			|| options.workModeToggle
			|| (image.spoiler && !options.spoilers)
			|| (image.fileType === fileTypes.gif && options.autogif)
		if (should) {
			view.renderImage(false)
		}
	}

	return false
}

// Add (You) to posts linking to the user's posts. Appends to array of posts,
// that might need to register a new reply to one of the user's posts.
function localizeLinks(post: Post, replies: number[]) {
	if (!post.links) {
		return
	}
	let el: HTMLElement
	for (let [id] of post.links) {
		if (!mine.has(id)) {
			continue
		}

		// Don't query DOM, until we know we need it
		if (!el) {
			el = post.view.el.querySelector("blockquote")
		}
		addYous(id, el)

		replies.push(id)
	}
}

function addYous(id: number, el: HTMLElement) {
	for (let a of el.querySelectorAll(`a[data-id="${id}"]`)) {
		a.textContent += " " + lang.posts["you"]
	}
}

// Add (You) to backlinks user's posts
function localizeBacklinks(post: Post) {
	if (!post.backlinks) {
		return
	}
	let el: HTMLElement
	for (let idStr in post.backlinks) {
		const id = parseInt(idStr)
		if (!mine.has(id)) {
			continue
		}

		// Don't query DOM, until we know we need it
		if (!el) {
			el = post.view.el.querySelector(".backlinks")
		}
		addYous(id, el)
	}
}

// Apply extra client-side localizations. Not done server-side for better
// cacheability.
export function localizeThreads() {
	localizeOmitted()
	if (posterName() || options.anonymise) {
		const name = posterName() || lang.posts["anon"]
		for (let el of threads.querySelectorAll(".name")) {
			el.textContent = name
		}
	} else if (options.lang !== "en_GB") { // Server renders in en_GB
		// Localize posts without a poster name or tripcode
		for (let el of threads.querySelectorAll(".name")) {
			if (el.textContent === "Anonymous") {
				el.textContent = lang.posts["anon"]
			}
		}

		// Localize banned post notices
		for (let el of threads.querySelectorAll(".banned")) {
			el.innerText = lang.posts["banned"]
		}
	}
}

// Localize omitted post and image span
function localizeOmitted() {
	// Server renders in en_GB
	if (options.lang === "en_GB") {
		return
	}
	for (let el of threads.querySelectorAll(".omit")) {
		if (!el) {
			continue
		}

		const posts = parseInt(el.getAttribute("data-omit")),
			images = parseInt(el.getAttribute("data-image-omit"))
		let text = pluralize(posts, lang.plurals["post"])
		if (images) {
			text += ` ${lang.posts["and"]} `
				+ pluralize(images, lang.plurals["image"])
		}
		text += ` ${lang.posts["omitted"]} `

		el.firstChild.replaceWith(text)
		el.querySelector("a.history").textContent = lang.posts["seeAll"]
	}
}

// If the post is still open, rerender its body, to sync the parser state.
// Needs to be done after models are populated to resolve temporary image links
// in open posts.
export function reparseOpenPosts() {
	for (let m of posts) {
		if (m.editing) {
			m.view.reparseBody()
		}
	}
}
