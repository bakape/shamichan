import { View } from "../base"
import { Post } from "./model"
import { getModel, mine } from "../state"
import { on, postJSON, HTML } from "../util"
import { FormView } from "../ui"
import lang from "../lang"
import { hidePost } from "./hide"
import { position } from "../mod"
import { ModerationLevel } from "../common"
import ReportForm from "./report"

interface ControlButton extends Element {
	_popup_menu: MenuView
}

// Spec for a single item of the drop down menu
type ItemSpec = {
	text: string
	keepOpen?: boolean // Keep open after click
	shouldRender: (m: Post) => boolean
	handler: (m: Post, el: Element) => void | Promise<void>
}

// Form embedded in the popup menu
abstract class MenuForm extends FormView {
	protected parentID: number;

	constructor(parent: Element, parentID: number, html: string,
		attrs: { [key: string]: any } = {},
	) {
		attrs["tag"] = "form"
		super(attrs);
		this.parentID = parentID;
		this.el.innerHTML = html
			+ HTML`
			<br>
			<input type="submit" value="${lang.ui["submit"]}">
			<input type="button" name="cancel" value="${lang.ui["cancel"]}">
			<div class="form-response admin"></div>`;
		parent.append(this.el);
	}

	protected closeMenu() {
		const el = this.el.closest(".control") as ControlButton;
		if (el && el._popup_menu) {
			el._popup_menu.remove();
		}
	}
}

// Form with one text field for submitting redirects
class RedirectForm extends MenuForm {
	private apiPath: string;

	constructor(parent: Element, parentID: number, apiPath: string) {
		super(parent, parentID,
			HTML`
			<br>
			<input type=text name=url placeholder="${lang.ui["location"]}">`);
		this.apiPath = apiPath;
	}

	protected async send() {
		let url = (this.el
			.querySelector("input[type=text]") as HTMLInputElement)
			.value;
		await postJSON(`/api/redirect/${this.apiPath}`, {
			id: this.parentID,
			url,
		});
		this.closeMenu();
		this.remove();
	}
}

// Actions to be performed by the items in the popup menu
const actions: { [key: string]: ItemSpec } = {
	hide: {
		text: lang.posts["hide"],
		shouldRender(m) {
			return !mine.has(m.id);
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
	redirectByThread: {
		text: lang.ui["redirectByThread"],
		keepOpen: true,
		shouldRender(m) {
			return position >= ModerationLevel.admin && m.id === m.op
		},
		handler(m, el) {
			new RedirectForm(el, m.id, "by-thread")
		},
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
		const act = actions[(e.target as Element).getAttribute('data-id')]
		if (act) {
			act.handler(this.model, (e.target as Element).closest("li"))
			if (!act.keepOpen) {
				this.remove()
			}
		}
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

export default () =>
	on(document, "click", openMenu, {
		passive: true,
		selector: ".control, .control svg, .control path",
	})
