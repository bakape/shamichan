// Post and image hover previews

import {emitChanges, ChangeEmitter} from "./model"
import View from "./view"
import {posts} from "./state"
import {hook} from "./hooks"
import options from "./options"
import {setAttrs} from "./util"

interface MouseMove extends ChangeEmitter {
	event: MouseEvent
}

const $overlay = document.querySelector("#hover-overlay")

// Currently displayed preview, if any
let postPreview: PostPreview,
	imagePreview: HTMLElement

// Centralised mousemove target tracking
// Logging only the target isn't a option because change:target doesn't seem
// to fire in some cases where the target is too similar for example changing
// between two post links (>>XXX) directly.
export const mouseMove = emitChanges<MouseMove>({
	event: {
		target: null,
	},
} as MouseMove)

// Bind mouse movement event listener
export default function bindMouseListener() {
	document.addEventListener(
		"mousemove",
		(event: MouseEvent) => {
			if (event.target !== mouseMove.event.target) {
				clear()
				mouseMove.event = event
			}
		},
		{passive: true},
	)
}

// Post hover preview view
class PostPreview extends View<any> {
	el: HTMLElement
	clickHandler: EventListener
	$parent: HTMLAnchorElement

	constructor(el: Element, parent: HTMLAnchorElement) {
		// Clone original element and modify
		const preview = el.cloneNode(true) as Element
		preview.removeAttribute("id")
		preview.classList.add("preview")

		super({el: preview})
		this.$parent = parent

		// Remove on parent click
		this.clickHandler = () =>
			this.remove()
		parent.addEventListener("click", this.clickHandler, {
			passive: true,
		})

		if ($overlay.firstChild) {
			$overlay.firstChild.remove()
		}
		$overlay.append(this.el)
		this.position()

		// TODO: Underline reference link in preview

	}

	// Position the preview element relative to it's parent link
	position() {
		const rect = this.$parent.getBoundingClientRect(),
			height = this.el.offsetHeight
		let left = rect.left,
			top = rect.top - height - 5

		// If post gets cut off at the top, put it bellow the link. The preview
		// will never take up more than 100% screen width, so no need for
		// checking horizontal overlflow.
		if (top < 0) {
			top += height + 23
		}

		this.el.style.left = left + "px"
		this.el.style.top = top + "px"
	}

	// Remove reference to this view from the parent element and module
	remove() {
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
		isWebm = /\.webm$/.test(src)

	// Nothing to preview for PDF or MP3
	const dontNeed =
		/\.pdf$/.test(src)
		|| /\.mp3$/.test(src)
		|| (isWebm && !options.webmHover)
	if (dontNeed) {
		return clear()
	}

	const el = document.createElement(isWebm ? "video" : "img")
	setAttrs(el, {
		src: src,
		autoplay: "",
		loop: "",
	})
	imagePreview = el
	$overlay.append(el)
}

function renderPostPreview(event: MouseEvent) {
	const target = event.target as HTMLAnchorElement
	if (!target.matches || !target.matches("a.history")) {
		return
	}
	const m = target.textContent.match(/^>>(\d+)/)
	if (!m) {
		return
	}
	const post = posts.get(parseInt(m[1]))
	if (!post) {

		// TODO: Try to fetch from API. This includes cross-thread posts

		return
	}
	postPreview = new PostPreview(post.view.el, target)
}

mouseMove.onChange("event", renderPostPreview)
mouseMove.onChange("event", renderImagePreview)

// Clear previews, when an image is expanded
hook("imageExpanded", clear)

