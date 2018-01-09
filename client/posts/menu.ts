import { View } from "../base"
import { Post } from "./model"
import { getModel } from "../state"
import { on } from "../util"
import lang from "../lang"
import { hidePost } from "./hide"
import { position, ModerationLevel } from "../mod"
import { postJSON } from "../util"
import CollectionView from "./collectionView"
import { PostData } from "../common"
import ReportForm from "./report"

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
	report: {
		text: lang.ui["report"],
		shouldRender(m) {
			return true
		},
		handler(m) {
			new ReportForm(m.id)
		},
	},
	viewSameIP: {
		text: lang.posts["viewBySameIP"],
		shouldRender: canModerateIP,
		async handler(m) {
			new CollectionView(await getSameIPPosts(m))
		},
	},
	deleteSameIP: {
		text: lang.posts["deleteBySameIP"],
		shouldRender: canModerateIP,
		async handler(m) {
			const posts = await getSameIPPosts(m)
			if (!posts) {
				return
			}
			if (!confirm(lang.ui["confirmDelete"])) {
				return
			}
			const res = await postJSON("/api/delete-post", posts.map(m =>
				m.id))
			if (res.status !== 200) {
				alert(await res.text())
			}
		},
	},
	toggleSticky: {
		text: lang.posts["toggleSticky"],
		shouldRender(m) {
			return position >= ModerationLevel.moderator && m.id === m.op
		},
		// Toggle sticky flag on a thread
		async handler(m) {
			const res = await postJSON("/api/sticky", {
				id: m.id,
				val: !m.sticky,
			})
			if (res.status !== 200) {
				return alert(await res.text())
			}
			m.sticky = !m.sticky
			m.view.renderSticky()
		},
	},
	toggleLock: {
		text: lang.ui["lockThread"],
		shouldRender(m) {
			return position >= ModerationLevel.moderator && m.id === m.op
		},
		async handler(m) {
			const res = await postJSON("/api/lock-thread", {
				id: m.id,
				val: !m.locked,
			})
			if (res.status !== 200) {
				return alert(await res.text())
			}
			m.locked = !m.locked
			m.view.renderLocked()
		},
	},
}

// Returns, if the post still likely has an IP attached and the client is
// logged in
function canModerateIP(m: Post): boolean {
	return position >= ModerationLevel.janitor
		&& m.time > Date.now() / 1000 - 24 * 7 * 3600
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
		this.model.view.el.classList.add("has-menu")
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
		const act = actions[(e.target as Element).getAttribute('data-id')]
		if (act) {
			act.handler(this.model)
			this.remove()
		}
	}

	// Also dereference from parent .control element
	public remove() {
		this.parent._popup_menu = null
		this.model.view.el.classList.remove("has-menu")
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

// Fetch posts with the same IP on this board
async function getSameIPPosts(m: Post): Promise<PostData[]> {
	const res = await postJSON(`/api/same-IP/${m.id}`, null)
	if (res.status !== 200) {
		alert(await res.text())
		return
	}
	return await res.json()
}

export default () =>
	on(document, "click", openMenu, {
		passive: true,
		selector: ".control, .control svg, .control svg path",
	})
