import { setBoardConfig, hidden, mine, posts, page } from "../state"
import options from "../options"
import { PostData, fileTypes } from "../common"
import { Post, PostView, hideRecursively } from "../posts"
import lang from "../lang"
import { postAdded, notifyAboutReply } from "../ui"
import { extractJSON } from "../util"

// Find board configurations in the HTML and apply them
export function extractConfigs() {
	setBoardConfig(extractJSON("board-configs"))
}

// Extract pregenerated rendered post data from DOM
export function extractPageData<T>(): {
	threads: T,
	backlinks: { [id: number]: { [id: number]: number } },
} {
	return {
		threads: extractJSON("post-data"),
		backlinks: extractJSON("backlink-data"),
	}
}

// Extract post model and view from the HTML fragment and apply client-specific
// formatting.
export function extractPost(
	post: PostData,
	op: number,
	board: string,
	backlinks: { [id: number]: { [id: number]: number } },
) {
	const el = document.getElementById(`p${post.id}`)
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

	// There are many client-side localizations for names, so best rerender
	// them all.
	view.renderName()

	localizeLinks(model)
	localizeBacklinks(model)
	postAdded(model)

	const { image } = model
	if (image) {
		if (options.hideThumbs
			|| options.workModeToggle
			|| (image.spoiler && !options.spoilers)
			|| (image.fileType === fileTypes.gif && options.autogif)
		) {
			view.renderImage(false)
		}
	}
}

// Add (You) to posts linking to the user's posts. Appends to array of posts,
// that might need to register a new reply to one of the user's posts.
function localizeLinks(post: Post) {
	if (!post.links) {
		return
	}
	let el: HTMLElement,
		isReply = false
	for (let id of new Set(post.links.map(l => l[0]))) {
		if (!mine.has(id)) {
			continue
		}
		isReply = true

		// Don't query DOM, until we know we need it
		if (!el) {
			el = post.view.el.querySelector("blockquote")
		}
		addYous(id, el)
	}
	if (isReply) {
		notifyAboutReply(post)
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

// Hide posts, that have been hidden (or linked hidden posts recursively, if
// enabled)
export function hidePosts() {
	for (let post of posts) {
		if (hidden.has(post.id)) {
			hideRecursively(post)
		}
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
