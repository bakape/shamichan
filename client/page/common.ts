import { setBoardConfig, hidden, mine, posts as postCollection } from "../state"
import options from "../options"
import { PostData, fileTypes, PostLink } from "../common"
import { Post, PostView } from "../posts"
import lang from "../lang"
import { notifyAboutReply } from "../ui"
import { threads, pluralize } from "../util"
import { posterName } from "../options"

// Find board configurations in the HTML and apply them
export function extractConfigs() {
	const conf = document.getElementById("board-configs").textContent
	setBoardConfig(JSON.parse(conf))
}

// Check if the rendered page is a ban page
export function isBanned(): boolean {
	return !!document.querySelector(".ban-page")
}

// Extract post model and view from the HTML fragment and apply client-specific
// formatting. Returns whether the element was removed.
export function extractPost(post: PostData, op: number): boolean {
	const el = document.getElementById(`p${post.id}`)
	if (hidden.has(post.id)) {
		el.remove()
		return true
	}
	post.op = op

	const model = new Post(post),
		view = new PostView(model, el)
	postCollection.add(model)

	// Apply client-specific formatting to a post rendered server-side

	// Render time-zone correction or relative time. Will do unneeded work,
	// if client is on UTC. Meh.
	view.renderTime()

	// Localize staff titles
	if (post.auth && options.lang != "en_GB") {
		view.renderName()
	}

	const {model: {links, backlinks, image}} = view
	localizeLinks(links, view, true)
	localizeLinks(backlinks, view, false)

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

// Add (You) to posts linking to the user's posts and trigger desktop
// notifications, if needed
function localizeLinks(links: PostLink[], view: PostView, notify: boolean) {
	if (!links) {
		return
	}
	for (let [id] of links) {
		if (!mine.has(id)) {
			continue
		}
		for (let el of view.el.querySelectorAll(`a[data-id="${id}"]`)) {
			// Can create doubles with circular quotes. Avoid that.
			if (!el.textContent.includes(lang.posts["you"])) {
				el.textContent += " " + lang.posts["you"]
			}
		}
		if (notify) {
			notifyAboutReply(view.model)
		}
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
	for (let m of postCollection) {
		if (m.editing) {
			m.view.reparseBody()
		}
	}
}
