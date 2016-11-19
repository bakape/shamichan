import AccountFormView, { newRequest, extractForm } from './common'
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
		const res = await postJSON(
			"/forms/configureServer",
			newRequest(),
		)
		if (res.status !== 200) {
			throw await res.text()
		}
		this.el.append(makeFrag(await res.text()))
		super.render()
	}

	// Extract and send the configuration struct from the form
	private async postConfigs() {
		const req = newRequest()
		extend(req, extractForm(this.el))
		this.postJSON("/admin/configureServer", req)
	}
}
