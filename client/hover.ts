// Post and image hover previews

import { emitChanges, ChangeEmitter } from "./model"
import View from "./view"
import { posts, getModel } from "./state"
import { hook } from "./hooks"
import options from "./options"
import { setAttrs, getClosestID } from "./util"
import { fetchJSON } from "./json"
import { PostData, Post } from "./posts/models"
import PostView from "./posts/view"

interface MouseMove extends ChangeEmitter {
	event: MouseEvent
}

const $overlay = document.querySelector("#hover-overlay")

// Currently displayed preview, if any
let postPreview: PostPreview,
	imagePreview: HTMLElement

// Centralised mousemove target tracking
const mouseMove = emitChanges<MouseMove>({
	event: {
		target: null,
	},
} as MouseMove)

// Post hover preview view
class PostPreview extends View<any> {
	el: HTMLElement
	clickHandler: EventListener
	observer: MutationObserver
	$parent: HTMLAnchorElement
	$source: HTMLElement

	constructor(el: HTMLElement, parent: HTMLAnchorElement) {
		super({ el: clonePost(el) })
		this.$parent = parent
		this.$source = el

		// Remove on parent click
		this.clickHandler = () =>
			this.remove()
		parent.addEventListener("click", this.clickHandler, {
			passive: true,
		})

		// Propagate post updates to clone
		this.observer = new MutationObserver(() =>
			this.renderUpdates())
		this.observer.observe(el, {
			childList: true,
			attributes: true,
			characterData: true,
			subtree: true,
		})

		this.render()
	}

	render() {
		// Underline reverse post links in preview
		const linksPost = ">>" + getClosestID(this.$parent)
		for (let el of this.el.querySelectorAll("a.history")) {
			if (!el.textContent.includes(linksPost)) {
				continue
			}
			el.classList.add("referenced")
		}

		if ($overlay.firstChild) {
			$overlay.firstChild.remove()
		}
		$overlay.append(this.el)
		this.position()
	}

	// Position the preview element relative to it's parent link
	position() {
		const rect = this.$parent.getBoundingClientRect()

		// The preview will never take up more than 100% screen width, so no
		// need for checking horizontal overlflow. Must be applied before
		// reading the height, so it takes into account post resizing to
		// viewport edge.
		this.el.style.left = rect.left + "px"

		const height = this.el.offsetHeight
		let top = rect.top - height - 5

		// If post gets cut off at the top, put it bellow the link
		if (top < 0) {
			top += height + 23
		}
		this.el.style.top = top + "px"
	}

	// Reclone and reposition on update. This is pretty expensive, but good
	// enough, because only one post will ever be previewied at a time
	renderUpdates() {
		const el = clonePost(this.$source)
		this.el.replaceWith(el)
		this.el = el as HTMLElement
		this.render()
	}

	// Remove reference to this view from the parent element and module
	remove() {
		this.observer.disconnect()
		this.$parent.removeEventListener("click", this.clickHandler)
		postPreview = null
		super.remove()
	}
}

// Clear any previews
function clear() {
	if (postPreview) {
		postPreview.remove()
		postPreview = null
	}
	if (imagePreview) {
		imagePreview.remove()
		imagePreview = null
	}
}

// Clone a post element as a preview
function clonePost(el: HTMLElement): HTMLElement {
	const preview = el.cloneNode(true) as HTMLElement
	preview.removeAttribute("id")
	preview.classList.add("preview")
	return preview
}

function renderImagePreview(event: MouseEvent) {
	if (!options.imageHover) {
		return
	}
	const target = event.target as HTMLElement
	if (target.tagName !== "IMG" || target.classList.contains("expanded")) {
		if (imagePreview) {
			imagePreview.remove()
			imagePreview = null
		}
		return
	}

	const link = target.closest("a")
	if (!link) {
		return
	}
	const src = link.getAttribute("href"),
		ext = src.slice(src.lastIndexOf(".") + 1)
	let tag: string

	switch (ext) {
		case "pdf": // Nothing to preview for PDF or MP3
		case "mp3":
			return clear()
		case "webm":
			if (!options.webmHover) {
				return clear()
			}
			tag = "video"
			break
		case "mp4":
		case "ogg":
			const model = getModel(link)
			// No video OGG are treated just like MP3
			if (!model || !model.image.video) {
				return clear()
			}
			tag = "video"
			break
		default:
			tag = "img"
	}

	const el = document.createElement(tag)
	setAttrs(el, {
		src: src,
		autoplay: "",
		loop: "",
	})
	imagePreview = el
	$overlay.append(el)
}

async function renderPostPreview(event: MouseEvent) {
	const target = event.target as HTMLAnchorElement
	if (!target.matches || !target.matches("a.history")) {
		return
	}
	const m = target.textContent.match(/^>{2,}(\d+)/)
	if (!m) {
		return
	}

	let post = posts.get(parseInt(m[1]))
	if (!post) {
		// Try to fetch from server, if this post is not currently displayed
		// due to lastN or in a different thread
		let data: PostData
		try {
			data = await fetchJSON<PostData>(`/json/post/${m[1]}`)
		} catch (e) {
			return
		}

		post = new Post(data)
		new PostView(post)
	}
	postPreview = new PostPreview(post.view.el, target)
}

// Bind mouse movement event listener
function onMouseMove(event: MouseEvent) {
	if (event.target !== mouseMove.event.target) {
		clear()
		mouseMove.event = event
	}
}

document.addEventListener("mousemove", onMouseMove, {
	passive: true,
})
mouseMove.onChange("event", renderPostPreview)
mouseMove.onChange("event", renderImagePreview)

// Clear previews, when an image is expanded
hook("imageExpanded", clear)

