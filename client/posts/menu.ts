import { View } from "../base"
import { Post } from "./model"
import { getModel, mine } from "../state"
import { on, postJSON, HTML } from "../util"
import { FormView } from "../ui"
import lang from "../lang"
import { hidePost } from "./hide"
import { position } from "../mod"
import CollectionView from "./collectionView"
import { PostData, ModerationLevel } from "../common"
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
			<input type=text name=url>`);
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

class DeleteByIPForm extends MenuForm {
	constructor(parent: Element, parentID: number) {
		let s = HTML`
		<hr>
		<span>${lang.ui["keepDeletingFor"]}</span>
		<br>
		<br>`;
		for (let id of ["day", "hour", "minute"]) {
			let label = lang.plurals[id][1];
			label = label[0].toUpperCase() + label.slice(1);
			s += HTML`
			<input type="number" name="${id}" min="0" placeholder="${label}">
			<br>`;
		}
		s += HTML`
		<input type="text" name="reason" class="full-width" placeholder="${lang.ui["reason"]}">
		<hr>`;
		super(parent, parentID, s);
		this.el.style.padding = "0.5em";

		// Reason is required, if duration set
		this.on("change", () => {
			const r = this.inputElement("reason");
			if (this.extractDuration() === 0) {
				r.removeAttribute("required");
			} else {
				r.setAttribute("required", "");
			}
		});
	}

	protected async send() {
		await postJSON("/api/delete-posts/by-ip", {
			id: this.parentID,
			duration: this.extractDuration(),
			reason: this.inputElement("reason").value,
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
		keepOpen: true,
		handler(m, el) {
			new DeleteByIPForm(el, m.id);
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
	redirectByIP: {
		text: lang.ui["redirectByIP"],
		keepOpen: true,
		shouldRender(m) {
			return position >= ModerationLevel.admin && likelyHasIP(m)
		},
		handler(m, el) {
			new RedirectForm(el, m.id, "by-ip")
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

// Returns, if the post still likely has an IP attached and the client is
// logged in
function canModerateIP(m: Post): boolean {
	return position >= ModerationLevel.janitor && likelyHasIP(m)
}

// Return, if post is fresh enough to likely not have its IP deleted yet
function likelyHasIP(m: Post): boolean {
	return m.time > Date.now() / 1000 - 24 * 7 * 3600
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
		selector: ".control, .control svg, .control path",
	})
