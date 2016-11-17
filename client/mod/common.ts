import { ui } from '../lang'
import FormView from "../forms"
import { ViewAttrs } from "../view"
import { accountPanel, loginID, sessionToken } from "./login"
import { write } from "../render"

// Common fields for authenticating `/admin` API request
export interface LoginCredentials {
	userID: string
	session: string
}

// Create a new base request for private logged in AJAX queries
export function newRequest<T extends LoginCredentials>(): T {
	return {
		userID: loginID,
		session: sessionToken,
	} as T
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
}
