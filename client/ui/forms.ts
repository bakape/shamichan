import { importTemplate, trigger } from '../util'
import { View, ViewAttrs, Model } from '../base'
import { config } from "../state";

export interface FormAttrs extends ViewAttrs {
	needCaptcha?: boolean
}

// Generic input form view with optional captcha support
abstract class FormView extends View<Model> {
	public el: HTMLFormElement
	private needCaptcha: boolean = false;

	protected abstract send(): void

	constructor(attrs: FormAttrs) {
		super(attrs)
		if (attrs.needCaptcha) {
			this.needCaptcha = true;
		}
		this.onClick({
			"input[name=cancel]": () =>
				this.remove(),
			".map-remove, .array-remove": e =>
				this.removeInput(e),
			".map-add": e =>
				this.addInput(e, "keyValue"),
			".array-add": e =>
				this.addInput(e, "arrayItem"),
		})
		this.on("submit", e =>
			this.submit(e))
	}


	// Submit form to server. Pass it to the assigned handler function
	private submit(event: Event) {
		event.preventDefault()
		if (config.captcha && this.needCaptcha) {
			// Prevents circular dependency
			trigger("renderCaptchaForm", this.send.bind(this));
		} else {
			this.send();
		}
	}

	// Render a text comment about the response status below the form
	protected renderFormResponse(text: string) {
		const el = this.el.querySelector(".form-response");
		if (el) {
			el.textContent = text;
		} else {
			alert(text);
		}
	}

	private addInput(event: Event, id: string) {
		(event.target as Element).before(importTemplate(id))
	}

	private removeInput(event: Event) {
		(event.target as Element).closest("span").remove()
	}
}

export default FormView
