import View from './view'
import Model from './model'
import {write, read} from './render'
import {config} from './state'
import {HTML, makeAttrs} from './util'
import {ui} from './lang'

// Data of a captcha challenge
export interface Captcha {
	captcha: string
	captchaID: string
}

// For generating unique IDs for every captcha
let captchaCounter = 0

// Returns a unique ID for captcha containers
export function newCaptchaID(): string {
	return `captcha-${captchaCounter++}`
}

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<Model> {
	widget: ACPuzzleController
	id: string

	constructor(id: string) {
		super({el: document.getElementById(id)})
		this.render()

		// Render the captcha widget only after the input field is focused
		read(() =>
			this.el
			.querySelector("input[name=adcopy_response]")
			.addEventListener("focus", () =>
				this.renderWidget()))

		this.onClick({
			".captcha-image img": () =>
				this.reload()
		})
	}

	// Render the container for the captcha
	render() {
		const {id} = this
		const imageAttrs: StringMap = {
			id: `adcopy-puzzle-image-${id}`,
			class: 'captcha-image',
			title: ui.reloadCaptcha,
		}
		const inputAttrs: StringMap = {
			id: `adcopy_response-${id}`,
			class: 'full-width',
			name: 'adcopy_response',
			type: "text",
			placeholder: ui.focusForCaptcha,
		}
		const html = HTML
			`<div id="adcopy-outer-${id}">
				<div ${makeAttrs(imageAttrs)}></div>
				<div id="adcopy-puzzle-audio-${id}" class="hidden"></div>
				<div id="adcopy-pixel-image-${id}" class="hidden"></div>
				<div>
					<span id="adcopy-instr-${id}" class="hidden"></span>
				</div>
				<input ${makeAttrs(inputAttrs)}>
				<input type="hidden" name="adcopy_challenge" id="adcopy_challenge-${id}">
				<a id="adcopy-link-refresh-${id}" class="hidden"></a>
				<a id="adcopy-link-audio-${id}" class="hidden"></a>
				<a id="adcopy-link-image-${id}" class="hidden"></a>
				<a id="adcopy-link-info-${id}" class="hidden"></a>
			</div>`
		write(() =>
			this.el.innerHTML = html)
	}

	// Render the actual captcha
	renderWidget() {
		this.widget = ACPuzzle.create(config.captchaPublicKey, this.id, {
			id: this.id,
			multi: true,
			theme: "custom",
		})
	}

	// Load a new captcha
	reload() {
		this.widget.reload()
	}

	remove() {
		if (this.widget) {
			write(() =>
				this.widget.destroy())
		}
		super.remove()
	}

	// Returns the data from the captcha widget
	data(): Captcha {
		return {
			captcha: this.widget.get_response(),
			captchaID: this.widget.get_challenge(),
		}
	}
}
