import {accountPannel} from './login'
import {write} from '../render'
import {FormView, FormViewAttrs} from '../forms'
import {mod as lang} from '../lang'
import {table, makeAttrs} from '../util'

// Specification for a single account management field
type FieldSpec = {
	type: string
	name: string
	maxLength: number
}

// Specs for all available account management fields
const fieldSpecs: {[key: string]: FieldSpec} = {
	id: {
		type: "text",
		name: "id",
		maxLength: 20,
	},
}

// Populate map with repeating password entires
for (let name of ["password", "repeat", "oldPassword", "newPassword"]) {
	fieldSpecs[name] = {
		name,
		type: "password",
		maxLength: 30,
	}
}

// Render account management input fields from specs
export function renderFields(...names: string[]): string {
	const fields = names.map(name =>
		fieldSpecs[name])
	return table(fields, ({type, name, maxLength}) => {
		const attrs = {
			type,
			name,
			maxlength: maxLength.toString() ,
			required: "",
		}
		return [
			`<label for="${name}">${lang[name]}:</label>`,
			`<input ${makeAttrs(attrs)}>`,
		]
	})
}

// Set a password match validator function for 2 input elements, that are
// children of the passed element.
export function validatePasswordMatch(
	parent: Element, name1: string, name2: string
) {
	const el1 = findInputEl(parent, name1),
		el2 = findInputEl(parent, name2)
	el2.onchange = () =>
		el2.setCustomValidity(el2.value !== el1.value ? lang.mustMatch : "")
}

// Find an input element by name within a parent form element
function findInputEl(parent: Element, name: string): HTMLInputElement  {
	return parent.querySelector(`input[name=${name}]`) as HTMLInputElement
}

// Generic input form that is embedded into AccountPanel
export default class AccountFormView extends FormView {
	constructor(attrs: FormViewAttrs, handler: () => void) {
		super(attrs, handler)
	}

	// Render a form field and embed the input fields inside it. Then append it
	// to the parrent view.
	renderForm(fields: string) {
		super.renderForm(fields)
		accountPannel.hideMenu()
		write(() =>
			accountPannel.el.append(this.el))
	}

	// Unhide the parent AccountPanel, when this view is removed
	remove() {
		super.remove()
		accountPannel.unhideMenu()
	}
}
