import { ui } from '../lang'
import FormView from "../forms"
import { ViewAttrs } from "../view"
import { accountPanel, loginID, sessionToken } from "./login"
import { write } from "../render"
import { postJSON } from "../fetch"

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
		el2 = findInputEl(parent, name2)
	const fn = () =>
		el2.setCustomValidity(el2.value !== el1.value ? ui.mustMatch : "")
	el1.onchange = el2.onchange = fn
}

// Find an input element by name within a parent form element
function findInputEl(parent: Element, name: string): HTMLInputElement {
	return parent.querySelector(`input[name=${name}]`) as HTMLInputElement
}

// Generic input form that is embedded into AccountPanel
export default class AccountFormView extends FormView {
	constructor(attrs: ViewAttrs, handler: () => void) {
		super(attrs, handler)
	}

	// Render a form field and embed the input fields inside it. Then append it
	// to the parent view.
	protected render() {
		accountPanel.toggleMenu(false)
		write(() =>
			accountPanel.el.append(this.el))
	}

	// Unhide the parent AccountPanel, when this view is removed
	public remove() {
		super.remove()
		accountPanel.toggleMenu(true)
	}

	// Send a POST request with a JSON body to the server and remove the view.
	// In case of errors, render them to the .form-response
	protected async postJSON(url: string, data: any) {
		const res = await postJSON(url, data)
		if (res.status !== 200) {
			this.renderFormResponse(await res.text())
			this.reloadCaptcha()
		} else {
			this.remove()
		}
	}
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
