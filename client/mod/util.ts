import {HTML, makeAttrs, makeEls} from '../util'
import {mod as lang, ui} from '../lang'
import View, {ViewAttrs} from '../view'
import AccountPanel from './login'
import {write, read} from '../render'
import Model from '../model'

export const enum inputType {
	boolean, number, string, select, multiline, map,
}

// Spec of a single input element for board and server control panels
export type InputSpec = {
	type: inputType
	name: string
	label?: string
	tooltip?: string
	value?: number|string|boolean|StringMap
	min?: number
	max?: number
	minLength?: number
	maxLength?: number
	choices?: string[]
	[index: string]: any
}

// Render a form input element for consumption by ../util.table
export function renderInput(spec: InputSpec): string[] {
	const attrs: StringMap = {
		name: spec.name,
		title: spec.tooltip,
	}

	switch (spec.type) {
	case inputType.boolean:
		attrs['type'] = 'checkbox'
		if (spec.value) {
			attrs["checked"] = ""
		}
		break
	case inputType.number:
		attrs["type"] = 'number'
		attrs['value'] = spec.value.toString()
		for (let prop of ['min', 'max']) {
			if (prop in spec) {
				attrs[prop] = spec[prop].toString()
			}
		}
		break
	case inputType.string:
		attrs["type"] = "text"
		attrs["value"] = spec.value as string
		break
	case inputType.select:
		return renderSelect(spec)
	case inputType.multiline:
		return renderTextArea(spec)
	case inputType.map:
		return renderMap(spec)
	}

	return [renderLabel(spec), `<input ${makeAttrs(attrs)}>`]
}

function renderSelect(spec: InputSpec): string[] {
	let html = `<select title="${spec.tooltip}" name="${spec.name}">`
	for (let item of spec.choices) {
		html += `<option value="${item}">${item}</option>`
	}
	html += "</select>"
	return [renderLabel(spec), html]
}

function renderTextArea(spec: InputSpec): string[] {
	const attrs: StringMap = {
		name: spec.name,
		title: spec.tooltip,
		rows: "3",
	}

	// Because textarea is a retardedly non-standard piece of shit that
	// can't even fucking support a fucking value attribute.
	read(() =>
		document
		.querySelector(`textarea[name=${spec.name}]`)
		.value = spec.value)

	return [
		renderLabel(spec),
		`<textarea ${makeAttrs(attrs)}></textarea>`,
	]
}

// Render a subform for assining map-like data
function renderMap(spec: InputSpec): string[] {
	let html = `<div name="${spec.name}" title="${spec.tooltip}">`
	const vals = spec.value
	for (let key in spec.value as StringMap) {
		html += renderKeyValuePair(key, (spec.value as StringMap)[key])
	}
	html += `<a class="map-add">${ui.add}</a><br></div>`

	return [renderLabel(spec), html]
}

// Render a single key-value input field pair in a map subform
function renderKeyValuePair(key: string, value: string): string {
	return HTML
		`<span>
			<input type="text" class="map-field" value=${key}>
			<input type="text" class="map-field" value=${value}>
			<a class="map-remove">
				[X]
			</a>
			<br>
		</span>`
}

function renderLabel(spec: InputSpec): string {
	return HTML
	`<label for="${spec.name}" title="${spec.tooltip}">
		${spec.label}:
	</label>
	<br>`
}

type FormHandler = (form: Element) => void

// Generic input form that is embedded into AccountPanel. Takes the parent
// AccountPanel view and function for extracting the form and sending the
// request as parameters.
export class FormView<M extends Model> extends View<M> {
	parent: AccountPanel
	handleForm: FormHandler // Function used for sending the form to the client

	constructor(attrs: ViewAttrs, handler: FormHandler) {
		super(attrs)
		this.parent = attrs.parent as AccountPanel
		this.handleForm = handler
		this.onClick({
			"input[name=cancel]": () =>
				this.remove(),
			".map-remove": e =>
				this.removeMapInput(e),
			".map-add": e =>
				this.addMapInput(e),
		})
		this.on('submit', e =>
			this.submit(e))
	}

	// Render a form field and embed the input fields inside it. Then append it
	// to the parrent view.
	renderForm(fields: string) {
		this.el.innerHTML = HTML
			`<form>
				${fields}
				<input type="submit" value="${lang.submit}">
				<input type="button" name="cancel" value="${ui.cancel}">
			</form>
			<div class="form-response admin"></div>`
		write(() => {
			this.parent.hideMenu()
			this.parent.el.append(this.el)
		})
	}

	// Submit form to server. Pass it to the assigned send function
	submit(event: Event) {
		event.preventDefault()
		this.handleForm(event.target as Element)
	}

	// Unhide the parent AccountPanel, when this view is removed
	remove() {
		super.remove()
		this.parent.unhideMenu()
	}

	// Render an additional map key-value input field pair
	addMapInput(event: Event) {
		write(() =>
			(event.target as Element)
				.before(...makeEls(renderKeyValuePair("", ""))))
	}

	// Remove a map key-vale input field pair
	removeMapInput(event: Event) {
		write(() =>
			(event.target as Element)
				.closest("span").remove())
	}
}
