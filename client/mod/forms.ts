import { inputValue, makeFrag, extend, postJSON } from '../util'
import { FormView } from "../ui"
import { validatePasswordMatch, newRequest } from './common'
import { View, ViewAttrs } from "../base"
import { accountPanel, loginID, reset } from "."
import lang from "../lang"

interface Removable {
	remove(): void
}

// Generic input form that is embedded into AccountPanel
abstract class AccountFormView extends FormView {
	constructor(attrs: ViewAttrs) {
		super(attrs)
	}

	// Render a form field and embed the input fields inside it. Then append it
	// to the parent view.
	protected render() {
		accountPanel.toggleMenu(false)
		accountPanel.el.append(this.el)
	}

	// Render a simple publically available form, that does not require to
	// submit any private information
	protected async renderPublicForm(url: string) {
		const res = await fetch(url)
		switch (res.status) {
			case 200:
				this.el.append(makeFrag(await res.text()))
				this.render()
				this.initCaptcha()
				break
			case 403:
				handle403(this)
				break
			default:
				throw await res.text()
		}
	}

	// Unhide the parent AccountPanel, when this view is removed
	public remove() {
		super.remove()
		accountPanel.toggleMenu(true)
	}

	// Send a POST request with a JSON body to the server and remove the view.
	// In case of errors, render them to the .form-response
	protected async postResponse(url: string, data: any) {
		const res = await postJSON(url, data)
		switch (res.status) {
			case 200:
				this.remove()
				break
			case 403:
				handle403(this)
				break
			default:
				this.renderFormResponse(await res.text())
				this.reloadCaptcha()
		}
	}
}

// Render the <select> for picking the owned board you want to manipulate
class OwnedBoardSelection extends View<null> {
	private parent: SelectedBoardForm

	constructor(parent: SelectedBoardForm) {
		super({ tag: "form" })
		this.parent = parent
		this.on("submit", e =>
			this.onSubmit(e))
		this.render()
	}

	private async render() {
		const res = await fetch(`/forms/ownedBoards/${loginID}`)
		switch (res.status) {
			case 200:
				this.el.append(makeFrag(await res.text()))
				this.parent.el.append(this.el)
				break
			case 403:
				handle403(this.parent)
				break
			default:
				throw await res.text()
		}
	}

	private onSubmit(e: Event) {
		e.preventDefault()
		e.stopPropagation()
		const board = ((e.target as Element)
			.querySelector("select") as HTMLInputElement)
			.value
		this.parent.renderNext(board)
		this.remove()
	}
}

// View that performs an action on a selected board
abstract class SelectedBoardForm extends AccountFormView {
	protected board: string
	protected boardSelector: OwnedBoardSelection

	public abstract renderNext(board: string): void

	constructor(attrs: ViewAttrs) {
		attrs.tag = "form"
		super(attrs)
		this.boardSelector = new OwnedBoardSelection(this)
		super.render()
	}
}

// View for changing a password
export class PasswordChangeView extends AccountFormView {
	constructor() {
		super({ tag: "form" })
		this.renderPublicForm("/forms/changePassword").then(() =>
			validatePasswordMatch(this.el, "newPassword", "repeat"))
	}

	protected send() {
		const req = newRequest()
		req["old"] = inputValue(this.el, "oldPassword")
		req["new"] = inputValue(this.el, "newPassword")
		this.injectCaptcha(req)
		this.postResponse("/admin/changePassword", req)
	}
}

// Board configuration panel
export class BoardConfigPanel extends SelectedBoardForm {
	constructor() {
		super({ class: "wide-fields" })
	}

	// Render the configuration input elements
	public async renderNext(board: string) {
		this.board = board

		const req = newRequest()
		req["id"] = board

		const res = await postJSON("/forms/configureBoard", req)
		switch (res.status) {
			case 200:
				const frag = makeFrag(await res.text())
				this.el.append(frag)
				break
			case 403:
				handle403(this)
				break
			default:
				throw await res.text()
		}
	}

	// Extract form data and send a request to apply the new configs
	protected send() {
		const req = newRequest()
		req["id"] = this.board
		extend(req, extractForm(this.el))

		// TODO: Some kind of form for inputting arrays
		req["eightball"] = req["eightball"].split("\n").slice(0, 100)

		this.postResponse("/admin/configureBoard", req)
	}
}

export class BoardDeletionView extends SelectedBoardForm {
	constructor() {
		super({})
	}

	public renderNext(board: string) {
		this.board = board
		this.renderPublicForm("/forms/captcha")
	}

	protected send() {
		const req = newRequest()
		req["id"] = this.board
		this.injectCaptcha(req)
		this.postResponse("/admin/deleteBoard", req)
	}
}

// Panel view for creating boards
export class BoardCreationPanel extends AccountFormView {
	constructor() {
		super({ tag: "form" })
		this.renderPublicForm("/forms/createBoard")
	}

	protected send() {
		const req = newRequest()
		req["name"] = inputValue(this.el, 'boardName')
		req["title"] = inputValue(this.el, 'boardTitle')
		this.injectCaptcha(req)

		this.postResponse("/admin/createBoard", req)
	}
}

// Panel for server administration controls such as global server settings
export class ConfigPanel extends AccountFormView {
	constructor() {
		super({
			tag: "form",
			class: "wide-fields", // The panel needs much larger text inputs
		})
		this.render()
	}

	// Request current configuration and render the panel
	protected async render() {
		const res = await postJSON("/forms/configureServer", newRequest())
		switch (res.status) {
			case 200:
				this.el.append(makeFrag(await res.text()))
				super.render()
				break
			case 403:
				handle403(this)
				break
			default:
				throw await res.text()
		}
	}

	// Extract and send the configuration struct from the form
	protected send() {
		const req = newRequest()
		extend(req, extractForm(this.el))
		this.postResponse("/admin/configureServer", req)
	}
}

// Reset any views and state on 403, which means an inconsistency between the
// client's assumptions about its permissions and the actual permissions stored
// in the database (likely because of session expiry).
function handle403(rem: Removable) {
	rem.remove()
	reset()
	alert(lang.ui["sessionExpired"])
}

// Extract values from an input form
export function extractForm(form: HTMLElement): { [key: string]: any } {
	const vals: { [key: string]: any } = {}

	const els = form
		.querySelectorAll("input[name], select[name], textarea[name]")
	for (let el of els as HTMLInputElement[]) {
		let val: any
		switch (el.type) {
			case "submit":
			case "button":
				continue
			case "checkbox":
				val = el.checked
				break
			case "number":
				val = parseInt(el.value)
				break
			default:
				val = el.value
		}
		vals[el.name] = val
	}

	// Read all key-value maps
	for (let map of form.querySelectorAll(".map-form")) {
		const fields = map.querySelectorAll(".map-field") as HTMLInputElement[]
		if (!fields.length) {
			continue
		}

		const m: { [key: string]: string } = {}
		for (let i = 0; i < fields.length; i += 2) {
			m[fields[i].value] = fields[i + 1].value
		}
		vals[map.getAttribute("name")] = m
	}

	return vals
}

