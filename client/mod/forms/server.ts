import { AccountForm } from "./common"
import { newRequest } from "../common"
import { postJSON, makeFrag } from "../../util"

// Panel for server administration controls such as global server settings
export class ServerConfigForm extends AccountForm {
	constructor() {
		super({
			tag: "form",
			class: "wide-fields", // The panel needs much larger text inputs
		})
		this.render()
	}

	// Request current configuration and render the panel
	protected async render() {
		const res = await postJSON("/forms/configureServer", newRequest())
		switch (res.status) {
			case 200:
				this.el.append(makeFrag(await res.text()))
				super.render()
				break
			case 403:
				this.handle403()
				break
			default:
				throw await res.text()
		}
	}

	// Extract and send the configuration struct from the form
	protected send() {
		this.postResponse("/admin/configureServer", req =>
			this.extractForm(req))
	}
}
