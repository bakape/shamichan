import View from './view'
import Model from './model'
import {write} from './render'
import {config} from './state'

// Solve Media AJAX API controller
// https://portal.solvemedia.com/portal/help/pub/ajax
interface ACPuzzleController  {
	create(
		key: string,
		elID: string,
		opts?: ACPuzzleOptions
	): ACPuzzleController
	destroy(): void
	get_challenge(): string
	get_response(): string
	reload(): void
}

interface ACPuzzleOptions {
	multi: boolean
	id: string
	theme: string
}

declare var ACPuzzle: ACPuzzleController

// Data of a captcha challenge
export interface Captcha {
	captcha: string
	captchaID: string
}

// For generating unique IDs for every captcha
let captchaCounter = 0

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<Model> {
	widget: ACPuzzleController
	id: string

	constructor(el: HTMLElement) {
		super({el})
		this.el.id = this.id = `captcha-${captchaCounter++}`
		; (this.el as HTMLElement).hidden = false
		this.render()

		// Render the captcha widget only after the input field is focused
		this.el
			.querySelector("input[name=adcopy_response]")
			.addEventListener("focus", () =>
				this.renderWidget())

		this.onClick({
			".captcha-image img": () =>
				this.reload()
		})
	}


	render() {
		// We need different IDs on all our elements because the spec is
		// retarded
		for (let el of this.el.querySelectorAll("*[data-id]")) {
			el.id = `${el.getAttribute("data-id")}-${this.id}`
		}

		// Reenable input fields
		for (let el of this.el.querySelectorAll("input")) {
			(el as HTMLInputElement).hidden = false
		}
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
