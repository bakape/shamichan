import FormView from "./forms";
import { hook } from "../util";
import { page } from "../state";

let instance: CaptchaForm;

const overlay = document.getElementById("captcha-overlay");

// Render a modal captcha input form
export function renderCaptchaForm(onSuccess: () => void) {
	if (!instance) {
		instance = new CaptchaForm(onSuccess);
	} else {
		instance.onSuccess = onSuccess;
		instance.focus();
	}
}

export const captchaLoaded = () => !!instance;

// Prevents circular dependency
hook("renderCaptchaForm", renderCaptchaForm);

// Floating captcha input modal
class CaptchaForm extends FormView {
	public onSuccess: () => void;

	constructor(onSuccess: () => void) {
		super({
			tag: "div",
			class: "modal glass",
			id: "captcha-form",
		});
		instance = this;
		this.onSuccess = onSuccess;
		this.render();
	}

	public remove() {
		instance = null;
		super.remove();
	}

	private async render() {
		overlay.prepend(this.el);
		const res = await fetch(
			`/api/captcha/${page.board}?${this.query({}).toString()}`)
		if (res.status !== 200) {
			this.renderFormResponse(await res.text());
			return;
		}
		const s = await res.text();
		this.el.innerHTML = s;
		this.el.style.margin = "auto";
		this.focus();
	}

	public focus() {
		const el = this.inputElement("captchouli-0");
		if (el) {
			el.focus();
		}
	}

	private query(d: { [key: string]: string }): URLSearchParams {
		d["captchouli-color"] = "inherit";
		d["captchouli-background"] = "inherit";
		return new URLSearchParams(d);
	}

	protected async send() {
		const body: { [key: string]: string } = {
			"captchouli-id": this.inputElement("captchouli-id").value,
		};
		for (let i = 0; i < 9; i++) {
			const k = `captchouli-${i}`;
			if (this.inputElement(k).checked) {
				body[k] = "on";
			}
		}

		const res = await fetch(`/api/captcha/${page.board}`, {
			body: this.query(body),
			method: "POST"
		});
		const t = await res.text();
		switch (res.status) {
			case 200:
				if (t !== "OK") {
					this.el.innerHTML = t;
					this.focus();
				} else {
					this.remove();
					this.onSuccess();
				}
				break;
			default:
				this.renderFormResponse(t);
		}
	}

	// Render a text comment about the response status below the form
	protected renderFormResponse(text: string) {
		this.el.querySelector("form").innerHTML = text;
		this.el.classList.add("admin");
	}
}
