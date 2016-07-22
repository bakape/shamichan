import View from './view'
import Model from './model'
import {write, read} from './render'
import {config} from './state'
import {HTML, makeAttrs} from './util'
import {ui} from './lang'

// For generating unique IDs for every captcha
let captchaCounter = 0

// Data of a captcha challenge
export interface Captcha {
	captcha: string
	captchaID: string
}

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<Model> {
	widget: ACPuzzleController
	id: string
	captchaID: number = captchaCounter++

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
		const id = this.captchaID.toString()
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
			id: this.captchaID.toString(),
			multi: true,
			theme: "custom",
		})
	}

	// Load a new captcha
	reload() {
		this.widget.reload()
	}

	remove() {
		write(() =>
			this.widget.destroy())
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
