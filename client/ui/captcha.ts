import { View } from '../base'
import { uncachedGET } from "../util"

// Wrapper around Solve Media's captcha service AJAX API
export default class CaptchaView extends View<null> {
	private captchaID: string
	private input: HTMLInputElement

	constructor(el: HTMLElement) {
		super({ el })

		// <noscript> loaded with AJAX can still load and cause submission
		// problems. Remove any.
		const ns = this.el.querySelector("noscript")
		if (ns) {
			ns.remove()
		}

		// Exposed outside through data() and therefore should always be defined
		this.input = this.el
			.querySelector(`input[name="captcha"]`) as HTMLInputElement

		this.render().catch(e => {
			alert(e)
			throw e
		})
	}

	// Render the actual captcha
	private async render() {
		// Hide before fetch to prevent popping
		const cID = this.inputElement("captchaID")
		cID.hidden = true

		const r = await uncachedGET(`/api/captcha/new`),
			text = await r.text()
		if (r.status !== 200) {
			throw text
		}
		this.captchaID = text;
		this.el
			.querySelector("img")
			.setAttribute("src", `/api/captcha/image/${this.captchaID}.png`)

		// Set captchaID, to enable sending with FormData()
		cID.value = this.captchaID
	}

	// Returns the data from the captcha widget
	public data(): { [key: string]: string } {
		// Captchas are disabled. Cache-induced race.
		if (!this.input) {
			return {}
		}

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
