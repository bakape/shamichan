import { importTemplate } from '../util'
import { View, ViewAttrs } from '../base'
import CaptchaView from './captcha'

interface FormAttrs extends ViewAttrs {
	lazyCaptcha?: boolean
}

// Generic input form view with optional captcha support
abstract class FormView extends View<null> {
	private captcha: CaptchaView

	protected abstract send(): void

	constructor(attrs: FormAttrs) {
		super(attrs)
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

		if (!attrs.lazyCaptcha) {
			this.initCaptcha()
		}
	}

	// Forms that are not rendered on initialization need to call this method
	// themselves
	public initCaptcha() {
		let captcha = this.el.querySelector(".captcha-container")
		if (captcha) {
			// Clear any previous captcha, when reusing form
			if (captcha.innerHTML !== "") {
				const el = document.createElement("div")
				el.classList.add("captcha-container")
				captcha.replaceWith(el)
				captcha = el
			}
			this.captcha = new CaptchaView(captcha)
		}
	}

	// Submit form to server. Pass it to the assigned handler function
	private submit(event: Event) {
		event.preventDefault()
		this.send()
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
			req["captcha"] = this.captcha.data()
		}
	}

	// Render a text comment about the response status below the form
	protected renderFormResponse(text: string) {
		this.el.querySelector(".form-response").textContent = text
	}

	// Load a new captcha, if present and response code is not 0
	public reloadCaptcha() {
		if (this.captcha) {
			this.captcha.reload()
		}
	}

	// Render an additional map key-value input field pair
	protected addMapInput(event: Event) {
		(event.target as Element).before(this.renderKeyValuePair("", ""))
	}

	// Render a single key-value input field pair in a map subform
	private renderKeyValuePair(key: string, val: string): DocumentFragment {
		const frag = importTemplate("keyValue"),
			[k, v] = frag.querySelectorAll("input")
		k.setAttribute("value", key)
		v.setAttribute("value", val)
		return frag
	}

	// Remove a map key-vale input field pair
	protected removeMapInput(event: Event) {
		(event.target as Element).closest("span").remove()
	}
}

export default FormView
