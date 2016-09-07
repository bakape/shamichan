// Post and image hover previews

import {emitChanges, ChangeEmitter} from "./model"
import View from "./view"
import {posts} from "./state"

interface MouseMove extends ChangeEmitter {
	event: MouseEvent
}

const $overlay = document.querySelector("#hover-overlay")

// Currently displayed preview, if any
let postPreview: PostPreview

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
				mouseMove.event = event
			}
		},
		{passive: true},
	)
}

mouseMove.onChange("event", (event: MouseEvent) => {
	if (postPreview) {
		postPreview.remove()
	}
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
})

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
