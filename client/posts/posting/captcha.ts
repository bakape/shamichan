import { FormView } from "../../ui";
import { postSM, postEvent } from ".";

let instance: CaptchaForm;

// Render a modal captcha input form
export function renderCaptchaForm() {
	if (!instance) {
		instance = new CaptchaForm();
	} else {
		instance.focus();
	}
}

// Floating captcha input modal
class CaptchaForm extends FormView {
	constructor() {
		super({
			tag: "form",
			class: "modal glass",
		});
		instance = this;
		this.render();
	}

	public remove() {
		instance = null;
		super.remove();
	}

	private async render() {
		const res = await fetch("/html/captcha")
		if (res.status !== 200) {
			this.renderFormResponse(await res.text());
			return;
		}
		this.el.innerHTML = await res.text();
		this.el.style.margin = "auto";
		this.el.style.display = "block";
		document.getElementById("modal-overlay").prepend(this.el);
		this.initCaptcha();
		this.focus();
	}

	public focus() {
		const el = this.inputElement("captcha");
		if (el) {
			el.focus();
		}
	}

	protected async send() {
		const data = this.captcha.data();
		const res = await fetch("/api/captcha", {
			headers: {
				'Content-Type': 'application/x-www-form-urlencoded',
			},
			body: `captchaID=${encodeURIComponent(data["captchaID"])}`
				+ `&captcha=${encodeURIComponent(data["solution"])}`,
			method: "POST"
		});
		if (res.status !== 200) {
			this.renderFormResponse(await res.text());
		} else {
			postSM.feed(postEvent.captchaSolved);
			this.remove();
		}
	}
}
