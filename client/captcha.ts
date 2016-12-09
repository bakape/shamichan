import View from './view'
import Model from './model'
import { write } from './render'
import { config } from './state'

// Solve Media AJAX API controller
// https://portal.solvemedia.com/portal/help/pub/ajax
interface ACPuzzleController {
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

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<Model> {
	public id: string
	private widget: ACPuzzleController

	constructor(el: HTMLElement) {
		super({ el })

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

	// Render the actual captcha
	private renderWidget() {
		const id = this.id.replace("captcha-", "")
		this.widget = ACPuzzle.create(config.captchaPublicKey, id, {
			id,
			multi: true,
			theme: "custom",
		})
	}

	// Load a new captcha
	public reload() {
		this.widget.reload()
	}

	public remove() {
		if (this.widget) {
			write(() =>
				this.widget.destroy())
		}
		super.remove()
	}

	// Returns the data from the captcha widget
	public data(): Captcha {
		return {
			captcha: this.widget.get_response(),
			captchaID: this.widget.get_challenge(),
		}
	}
}
