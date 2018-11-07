import { FormView } from "../ui"
import { makeFrag } from "../util"

// Modal for submitting reports
export default class ReportForm extends FormView {
	public el: HTMLFormElement

	constructor(id: number) {
		super({
			tag: "form",
			class: "modal glass show report-form",
			needCaptcha: true,
		})
		this.render(id)
	}

	private async render(id: number) {
		const res = await fetch(`/html/report/${id}`),
			t = await res.text()

		document.getElementById("modal-overlay").prepend(this.el)

		switch (res.status) {
			case 200:
				this.el.append(makeFrag(t))
				this.inputElement("reason").focus()
				break
			default:
				this.renderFormResponse(t)
		}
	}

	protected async send() {
		const res = await fetch("/api/report", {
			method: "POST",
			body: new FormData(this.el),
		})
		if (res.status !== 200) {
			this.renderFormResponse(await res.text())
		} else {
			this.remove()
		}
	}
}
