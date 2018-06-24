import lang from '../lang'
import { getCookie, inputElement } from "../util"

// Returns current login ID in use
export function loginID(): string {
	return getCookie("loginID")
}

// Returns current login session token in use
export function sessionToken(): string {
	return getCookie("session")
}

// Set a password match validator function for 2 input elements, that are
// children of the passed element.
export function validatePasswordMatch(
	parent: Element, name1: string, name2: string
) {
	const el1 = inputElement(parent, name1),
		el2 = inputElement(parent, name2)
	el1.onchange = el2.onchange = () => {
		const s = el2.value !== el1.value ? lang.ui["mustMatch"] : ""
		el2.setCustomValidity(s)
	}
}
