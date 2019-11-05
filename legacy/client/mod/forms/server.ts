import { AccountForm } from "./common"
import { makeFrag } from "../../util"

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
		const res = await fetch("/html/configure-server", {
			method: "POST",
			credentials: "include",
		});
		const body = await res.text();
		switch (res.status) {
			case 200:
				this.el.append(makeFrag(body))
				super.render()
				break
			case 403:
				this.handle403(body);
				break
			default:
				throw body;
		}
	}

	// Extract and send the configuration struct from the form
	protected send() {
		this.postResponse("/api/configure-server", req =>
			this.extractForm(req))
	}
}
