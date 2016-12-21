import lang from '../lang'
import FormView from "../forms"
import { ViewAttrs } from "../view"
import { accountPanel, loginID, sessionToken, reset } from "./login"
import { write } from "../render"
import { postJSON } from "../fetch"
import { makeFrag } from "../util"
import View from "../view"

interface Removable {
	remove(): void
}

// Create a new base request for private logged in AJAX queries
export function newRequest(): { [key: string]: any } {
	return {
		userID: loginID,
		session: sessionToken,
	}
}

// Set a password match validator function for 2 input elements, that are
// children of the passed element.
export function validatePasswordMatch(
	parent: Element, name1: string, name2: string
) {
	const el1 = findInputEl(parent, name1),
		el2 = findInputEl(parent, name2),
		v = el2.value !== el1.value ? lang.ui["mustMatch"] : ""
	el1.onchange = el2.onchange = () =>
		el2.setCustomValidity(v)
}

// Find an input element by name within a parent form element
function findInputEl(parent: Element, name: string): HTMLInputElement {
	return parent.querySelector(`input[name=${name}]`) as HTMLInputElement
}

// Generic input form that is embedded into AccountPanel
export abstract class AccountFormView extends FormView {
	constructor(attrs: ViewAttrs) {
		super(attrs)
	}

	// Render a form field and embed the input fields inside it. Then append it
	// to the parent view.
	protected render() {
		accountPanel.toggleMenu(false)
		write(() =>
			accountPanel.el.append(this.el))
	}

	// Render a simple publically available form, that does not require to
	// submit any private information
	protected async renderPublicForm(url: string) {
		const res = await fetch(url)
		switch (res.status) {
			case 200:
				this.el.append(makeFrag(await res.text()))
				this.render()
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

// Reset any views and state on 403, which means an inconsistency between the
// client's assumptions about its permissions and the actual permissions stored
// in the database (likely because of session expiry).
export function handle403(rem: Removable) {
	rem.remove()
	reset()
	alert(lang.ui["sessionExpired"])
}

// Extract values from an input form
export function extractForm(form: HTMLElement): { [key: string]: any } {
	const vals: { [key: string]: any } = {}

	const els = form.querySelectorAll(
		"input[name], select[name], textarea[name]",
	)
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

// Render the <select> for picking the owned board you want to manipulate
export class OwnedBoardSelection extends View<null> {
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
				write(() =>
					this.parent.el.append(this.el))
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
export abstract class SelectedBoardForm extends AccountFormView {
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
