import View from "../view"
import { Post } from "./models"
import { getModel, mine } from "../state"
import { threads, write } from "../render"
import { on, outerWidth } from "../util"
import * as lang from "../lang"
import { hidePost } from "./hide"
import { spoilerImage } from "./posting/upload"

interface ControlButton extends Element {
	_popup_menu: MenuView
}

// Spec for a single item of the dropdown menu
type ItemSpec = {
	text: string
	shouldRender: (m: Post) => boolean
	handler: (m: Post) => void|Promise<void>
}

// Actions to be performed by the items in the popup menu
const actions: { [key: string]: ItemSpec } = {
	hide: {
		text: lang.images.hide,
		shouldRender(m) {
			return true
		},
		handler: hidePost,
	},
	spoiler: {
		text: lang.posts.spoiler,
		shouldRender({id, image}) {
			return !!image && !image.spoiler && mine.has(id)
		},
		handler: spoilerImage,
	},
}

// Post header drop down menu
class MenuView extends View<Post> {
	el: HTMLElement
	parent: ControlButton

	constructor(parent: ControlButton, model: Post) {
		super({
			model,
			tag: "ul",
			class: "popup-menu glass",
		})
		this.parent = parent
		parent._popup_menu = this
		this.render()
		this.on("click", e => this.handleClick(e), {
			passive: true,
		})
	}

	render() {
		for (let key in actions) {
			const {shouldRender, text} = actions[key]
			if (!shouldRender(this.model)) {
				continue
			}
			const li = document.createElement("li")
			li.setAttribute("data-id", key)
			li.textContent = text
			this.el.append(li)
		}

		const {el, parent} = this
		write(() =>
			parent.append(el))

		// Calculate position. Can't use CSS translate, because it shifts
		// the background.
		el.style.left =
			el.getBoundingClientRect().left
			- (outerWidth(el) + el.offsetWidth) * 0.6
			+ 'px'
	}

	// Run appropriate handler on click or simply remove the menu
	handleClick(e: Event) {
		actions[(e.target as Element).getAttribute('data-id')]
			.handler(this.model)
		this.remove()
	}

	// Also dereference from parent .control element
	remove() {
		this.parent._popup_menu = null
		super.remove()
	}
}

// Open a popup menu, after clicking on a .control down arrow
function openMenu(e: Event) {
	const parent = (e.target as Element).closest(".control") as ControlButton

	if (parent._popup_menu) {
		return parent._popup_menu.remove()
	}

	const model = getModel(parent)
	if (model) {
		new MenuView(parent, model)
	}
}

export default function bind() {
	on(threads, "click", openMenu, {
		passive: true,
		selector: ".control, .control svg, .control svg path",
	})
}
