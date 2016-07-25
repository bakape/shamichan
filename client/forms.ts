// Utility functions and classes for rendering forms

import {HTML, makeAttrs, makeEls, extend} from './util'
import View, {ViewAttrs} from './view'
import Model from './model'
import {write, read} from './render'
import {ui} from './lang'
import {config} from './state'
import CaptchaView from './captcha'

export const enum inputType {
	boolean, number, string, select, multiline, map,
}

// Spec of a single input element for board and server control panels
export type InputSpec = {
	type: inputType
	name: string
	label?: string
	tooltip?: string
	pattern?: string
	value?: number|string|boolean|StringMap
	min?: number
	max?: number
	maxLength?: number
	choices?: string[]
	[index: string]: any
}

export type FormHandler = (form: Element) => void

export interface FormViewAttrs extends ViewAttrs {
	noCaptcha?: boolean
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
		if (spec.value !== undefined) {
			attrs['value'] = spec.value.toString()
		}
		for (let prop of ['min', 'max']) {
			if (prop in spec) {
				attrs[prop] = spec[prop].toString()
			}
		}
		break
	case inputType.string:
		attrs["type"] = "text"
		attrs["value"] = spec.value as string || ""
		if (spec.pattern) {
			attrs["pattern"] = spec.pattern
		}
		if ("maxLength" in spec) {
			attrs["maxlength"] = spec.maxLength.toString()
		}
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
	if (spec.value) {
		for (let key in spec.value as StringMap) {
			html += renderKeyValuePair(key, (spec.value as StringMap)[key])
		}
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

// Generic input form view with optional captcha support
export class FormView extends View<Model> {
	handleForm: FormHandler // Function used for sending the form to the client
	captcha: CaptchaView
	noCaptcha: boolean

	constructor(attrs: FormViewAttrs, handler: FormHandler) {
		super(attrs)
		this.handleForm = handler
		this.noCaptcha = attrs.noCaptcha
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

	// Render a form field and embed the input fields inside it
	renderForm(fields: string) {
		const captchaID = this.id + "-captcha"

		this.el.innerHTML = HTML
			`<form>
				${fields}
				<div id="${captchaID}"></div>
				<input type="submit" value="${ui.submit}">
				<input type="button" name="cancel" value="${ui.cancel}">
			</form>
			<div class="form-response admin"></div>`
		write(() => {
			if (config.captcha && !this.noCaptcha) {
				this.captcha = new CaptchaView(captchaID)
			}
		})
	}

	// Submit form to server. Pass it to the assigned handler function
	submit(event: Event) {
		event.preventDefault()
		this.handleForm(event.target as Element)
	}

	// Also destroy captcha, if any
	remove() {
		if (this.captcha) {
			this.captcha.remove()
		}
		super.remove()
	}

	// Inject captcha data into the request struct, if any
	injectCaptcha(req: {}) {
		if (this.captcha) {
			extend(req, this.captcha.data())
		}
	}

	// Load a new captcha, if present and response code is not 0
	reloadCaptcha(code: number) {
		if (code !== 0 && this.captcha) {
			this.captcha.reload()
		}
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
