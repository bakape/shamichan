// Utility functions and classes for rendering forms

import { HTML, makeAttrs, makeFrag, extend } from './util'
import View from './view'
import Model from './model'
import { write, read } from './render'
import { ui } from './lang'
import CaptchaView from './captcha'

type StringMap = { [key: string]: string }

export const enum inputType { boolean, number, string, select, multiline, map }

// Spec of a single input element for board and server control panels
export type InputSpec = {
	type: inputType
	required?: boolean
	name: string
	label?: string
	placeholders?: boolean // Render placeholders inside input elements
	tooltip?: string
	pattern?: string
	value?: number | string | boolean | StringMap
	min?: number
	max?: number
	maxLength?: number
	rows?: number
	choices?: string[]
	[index: string]: any
}

// Render a multiline input textarea
export function renderTextArea(spec: InputSpec, attrs: StringMap): [string, string] {
	attrs["rows"] = (spec.rows || 3).toString()
	if ("maxLength" in spec) {
		attrs["maxlength"] = spec.maxLength.toString()
	}

	// Because textarea is a retardedly non-standard piece of shit that
	// can't even fucking support a fucking value attribute.
	if (spec.value) {
		read(() =>
			(document
				.querySelector(`textarea[name=${spec.name}]`) as HTMLInputElement)
				.value = spec.value as string)
	}

	return [
		renderLabel(spec),
		`<textarea ${makeAttrs(attrs)}></textarea>`,
	]
}

// Render a subform for assigning map-like data
export function renderMap(spec: InputSpec): [string, string] {
	let html = `<div name="${spec.name}" title="${spec.tooltip || ""}">`
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
		`<label for="${spec.name}" title="${spec.tooltip || ""}">
		${spec.label}:
	</label>
	<br>`
}

// Generic input form view with optional captcha support
export class FormView extends View<Model> {
	private handleForm: () => void // Function used for sending the form to the client
	private captcha: CaptchaView

	constructor(el: HTMLElement, handler: () => void) {
		super({ el })
		this.handleForm = handler
		this.onClick({
			"input[name=cancel]": () =>
				this.remove(),
			".map-remove": e =>
				this.removeMapInput(e),
			".map-add": e =>
				this.addMapInput(e),
		})
		this.on("submit", e =>
			this.submit(e))

		const captcha = this.el.querySelector(".captcha-container")
		if (captcha) {
			this.captcha = new CaptchaView(captcha)
		}
	}

	// Submit form to server. Pass it to the assigned handler function
	private submit(event: Event) {
		event.preventDefault()
		this.handleForm()
	}

	// Also destroy captcha, if any
	public remove() {
		if (this.captcha) {
			this.captcha.remove()
		}
		super.remove()
	}

	// Inject captcha data into the request struct, if any
	protected injectCaptcha(req: {}) {
		if (this.captcha) {
			extend(req, this.captcha.data())
		}
	}

	// Render a text comment about the response status below the form
	protected renderFormResponse(text: string) {
		write(() =>
			this.el.querySelector(".form-response").textContent = text)
	}

	// Load a new captcha, if present and response code is not 0
	protected reloadCaptcha(code: number) {
		if (code !== 0 && this.captcha) {
			this.captcha.reload()
		}
	}

	// Render an additional map key-value input field pair
	protected addMapInput(event: Event) {
		write(() =>
			(event.target as Element)
				.before(makeFrag(renderKeyValuePair("", ""))))
	}

	// Remove a map key-vale input field pair
	protected removeMapInput(event: Event) {
		write(() =>
			(event.target as Element)
				.closest("span").remove())
	}
}
