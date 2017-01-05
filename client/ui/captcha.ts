import { View } from '../base'
import { config } from '../state'
import options from "../options"

// Google's Recaptcha JS API
interface GRecaptcha {
	render(container: Element, parameters: RenderParams): string
	reset(id: string): void
	getResponse(id: string): string
}

interface RenderParams {
	sitekey: string
	theme: Theme
	size: "compact" | "normal"
}

type Theme = "light" | "dark"

declare var grecaptcha: GRecaptcha

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<null> {
	private widgetID: string

	constructor(el: HTMLElement) {
		super({ el })
		this.render()
	}

	// Render the actual captcha
	private render() {
		let theme: Theme
		switch (options.theme) {
			case "ashita":
			case "console":
			case "glass":
			case "higan":
			case "inumi":
			case "mawaru":
			case "ocean":
				theme = "dark"
				break
			default:
				theme = "light"
		}

		this.widgetID = grecaptcha.render(this.el, {
			sitekey: config.captchaPublicKey,
			theme,
			size: "normal",
		})
	}

	// Load a new captcha
	public reload() {
		grecaptcha.reset(this.widgetID)
	}

	// Returns the data from the captcha widget
	public data(): { [key: string]: string } {
		return { captcha: grecaptcha.getResponse(this.widgetID) }
	}
}
