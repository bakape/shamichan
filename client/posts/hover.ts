// Post and image hover previews

import { posts, getModel } from "../state"
import options from "../options"
import {
	setAttrs, getClosestID, fetchJSON, hook, emitChanges, ChangeEmitter
} from "../util"
import { Post } from "./model"
import ImageHandler from "./images"
import PostView from "./view"
import { PostData } from "../common"

interface MouseMove extends ChangeEmitter {
	event: MouseEvent
}

const overlay = document.querySelector("#hover-overlay")

// Currently displayed preview, if any
let postPreview: PostPreview,
	imagePreview: HTMLElement

// Centralized mousemove target tracking
const mouseMove = emitChanges<MouseMove>({
	event: {
		target: null,
	},
} as MouseMove)

// Post hover preview view
class PostPreview extends ImageHandler {
	public el: HTMLElement
	private clickHandler: EventListener
	private observer: MutationObserver
	private parent: HTMLElement
	private source: HTMLElement
	private sourceModel: Post

	constructor(model: Post, parent: HTMLElement) {
		const {el} = model.view
		super({ el: clonePost(el) })
		this.parent = parent
		this.model = Object.assign({}, model)
		this.sourceModel = model
		this.source = el

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

	private render() {
		// Remove any inline expanded posts
		for (let el of this.el.querySelectorAll("article")) {
			el.remove()
		}

		// Remove any existing reverse post link highlights due to link inline
		// expansion
		for (let el of this.el.querySelectorAll("a.history.referenced")) {
			el.classList.remove("referenced")
		}

		// Underline reverse post links in preview
		const patt = new RegExp(`[>\/]` + getClosestID(this.parent))
		for (let el of this.el.querySelectorAll("a.history")) {
			if (!patt.test(el.textContent)) {
				continue
			}
			el.classList.add("referenced")
		}

		// Contract any expanded open thumbnails
		const img = this.sourceModel.image
		if (img && img.expanded) {
			// Clone parent model's image and render contracted thumbnail
			this.model.image = Object.assign({}, this.sourceModel.image)
			this.contractImage(null, false)
		}

		const fc = overlay.firstChild
		if (fc !== this.el) {
			if (fc) {
				fc.remove()
			}
			overlay.append(this.el)
		}

		this.position()

		// Highlight target post, if present
		this.sourceModel.view.setHighlight(true)
	}

	// Position the preview element relative to it's parent link
	private position() {
		const rect = this.parent.getBoundingClientRect()

		// The preview will never take up more than 100% screen width, so no
		// need for checking horizontal overflow. Must be applied before
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
	// enough, because only one post will ever be previewed at a time
	private renderUpdates() {
		const el = clonePost(this.source)
		this.el.replaceWith(el)
		this.el = el
		this.render()
	}

	// Remove reference to this view from the parent element and module
	public remove() {
		this.observer.disconnect()
		this.parent.removeEventListener("click", this.clickHandler)
		postPreview = null
		super.remove()
		this.sourceModel.view.setHighlight(false)
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
	const bypass = target.tagName !== "IMG"
		|| target.classList.contains("expanded")
		|| target.classList.contains("catalog")
	if (bypass) {
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
		case "zip":
		case "7z":
		case "gz":
		case "xz":
			return clear()
		case "webm":
			if (!options.webmHover) {
				return clear()
			}
			tag = "video"
			break
		case "mp4":
		case "ogg":
			if (!options.webmHover) {
				return clear()
			}
			const model = getModel(link)
			// No video OGG and MP4 are treated just like MP3
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
	overlay.append(el)
}

async function renderPostPreview(event: MouseEvent) {
	let target = event.target as HTMLElement
	if (!target.matches || !target.matches("a.history, .hash-link")) {
		return
	}
	if (target.classList.contains("hash-link")) {
		target = target.previousElementSibling as HTMLElement
	}
	if (target.matches("em.expanded > a")) {
		return
	}
	const id = parseInt(target.getAttribute("data-id"))
	if (!id) {
		return
	}

	let post = posts.get(id)
	if (!post) {
		// Try to fetch from server, if this post is not currently displayed
		// due to lastN or in a different thread
		const [data] = await fetchJSON<PostData>(`/json/post/${id}`)
		if (data) {
			post = new Post(data)
			new PostView(post, null)
		} else {
			return
		}
	}
	postPreview = new PostPreview(post, target)
}

// Bind mouse movement event listener
function onMouseMove(event: MouseEvent) {
	if (event.target !== mouseMove.event.target) {
		clear()
		mouseMove.event = event
	}
}

export default () => {
	document.addEventListener("mousemove", onMouseMove, {
		passive: true,
	})
	mouseMove.onChange("event", renderPostPreview)
	mouseMove.onChange("event", renderImagePreview)

	// Clear previews, when an image is expanded
	hook("imageExpanded", clear)
}

