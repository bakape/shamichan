import AccountFormView, { newRequest, extractForm, handle403 } from './common'
import { makeFrag, extend } from '../util'
import { postJSON } from "../fetch"

// Panel for server administration controls such as global server settings
export default class ConfigPanel extends AccountFormView {
	constructor() {
		const attrs = {
			tag: "form",
			class: "wide-fields", // The panel needs much larger text inputs
		}
		super(attrs, () =>
			this.postConfigs())
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
				handle403(this)
				break
			default:
				throw await res.text()
		}
	}

	// Extract and send the configuration struct from the form
	private async postConfigs() {
		const req = newRequest()
		extend(req, extractForm(this.el))
		this.postResponse("/admin/configureServer", req)
	}
}
