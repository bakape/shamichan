import lang from '../lang'
import { loginID, sessionToken } from "."

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
