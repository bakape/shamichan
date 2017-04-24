import { View } from '../base'
import { uncachedGET } from "../util"

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<null> {
	private captchaID: string
	private image: HTMLImageElement
	private input: HTMLInputElement

	constructor(el: HTMLElement) {
		super({ el })

		// <noscript> loaded with AJAX can still load and cause submission
		// problems. Remove any.
		const ns = this.el.querySelector("noscript")
		if (ns) {
			ns.remove()
		}

		this.render().catch(e => {
			alert(e)
			throw e
		})
	}

	// Render the actual captcha
	private async render() {
		const r = await uncachedGET(`/captcha/new`),
			text = await r.text()
		if (r.status !== 200) {
			throw text
		}
		this.captchaID = text
		this.image = this.el.querySelector("img") as HTMLImageElement
		this.image.setAttribute("src", `/captcha/image/${this.captchaID}.png`)
		this.input = this.el
			.querySelector(`input[name="captcha"]`) as HTMLInputElement
	}

	// Returns the data from the captcha widget
	public data(): { [key: string]: string } {
		return {
			captchaID: this.captchaID,
			solution: this.input.value,
		}
	}

	// Load a new captcha
	public reload() {
		this.input.value = ""
		this.render()
	}
}
