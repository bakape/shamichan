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

interface Window {
	onRecaptchaLoad: () => void
}

declare var window: Window

let scriptLoaded: Promise<void>,
	loadingScript = false

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<null> {
	private widgetID: string

	constructor(el: HTMLElement) {
		super({ el })
		this.render().catch(err =>
			alert("The tin foil is too far up your ass. Stop blocking captchas, faggot."))
	}

	// Render the actual captcha
	private async  render() {
		if (!loadingScript) {
			this.loadScript()
		}
		await scriptLoaded

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
	public data(): string {
		return grecaptcha.getResponse(this.widgetID)
	}

	// Load the grecaptcha script from Google's servers
	private loadScript() {
		loadingScript = true
		const el = document.createElement("script")
		scriptLoaded = new Promise<void>((resolve, reject) => {
			window.onRecaptchaLoad = resolve
			el.onerror = reject
		})
		el.src = "https://www.google.com/recaptcha/api.js?onload=onRecaptchaLoad&render=explicit"
		document.head.append(el)
	}
}
