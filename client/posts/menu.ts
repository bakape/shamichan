import { View } from "../base"
import { Post } from "./model"
import { getModel } from "../state"
import { on } from "../util"
import lang from "../lang"
import { hidePost } from "./hide"
import { loginID } from "../mod"
import { postJSON } from "../util"
import CollectionView from "./collectionView"

interface ControlButton extends Element {
	_popup_menu: MenuView
}

// Spec for a single item of the drop down menu
type ItemSpec = {
	text: string
	shouldRender: (m: Post) => boolean
	handler: (m: Post) => void | Promise<void>
}

// Actions to be performed by the items in the popup menu
const actions: { [key: string]: ItemSpec } = {
	hide: {
		text: lang.posts["hide"],
		shouldRender(m) {
			return true
		},
		handler: hidePost,
	},
	viewSameIP: {
		text: lang.posts["viewBySameIP"],
		shouldRender(m) {
			return !!loginID() && m.time > Date.now() / 1000 - 24 * 7 * 3600
		},
		handler: getSameIPPosts,
	},
	toggleSticky: {
		text: lang.posts["toggleSticky"],
		shouldRender(m) {
			return !!loginID() && m.id === m.op
		},
		handler: toggleSticky,
	},
}

// Post header drop down menu
class MenuView extends View<Post> {
	public el: HTMLElement
	private parent: ControlButton

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

	private render() {
		for (let key in actions) {
			const { shouldRender, text } = actions[key]
			if (!shouldRender(this.model)) {
				continue
			}
			const li = document.createElement("li")
			li.setAttribute("data-id", key)
			li.textContent = text
			this.el.append(li)
		}
		this.parent.append(this.el)
	}

	// Run appropriate handler on click or simply remove the menu
	private handleClick(e: Event) {
		actions[(e.target as Element).getAttribute('data-id')]
			.handler(this.model)
		this.remove()
	}

	// Also dereference from parent .control element
	public remove() {
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

// Fetch and render posts with the same IP on this board
async function getSameIPPosts(m: Post) {
	const res = await postJSON("/admin/sameIP", {
		board: m.board,
		id: m.id,
	})
	if (res.status !== 200) {
		return alert(await res.text())
	}
	new CollectionView(await res.json())
}

// Toggle sticky flag on a thread
async function toggleSticky(m: Post) {
	const res = await postJSON("/admin/sticky", {
		sticky: !m.sticky,
		id: m.id,
	})
	if (res.status !== 200) {
		return alert(await res.text())
	}
	m.sticky = !m.sticky
	m.view.renderSticky()
}

export default () =>
	on(document, "click", openMenu, {
		passive: true,
		selector: ".control, .control svg, .control svg path",
	})
