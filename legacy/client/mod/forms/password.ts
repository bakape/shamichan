import { AccountForm } from "./common"
import { validatePasswordMatch } from "../common"

// View for changing a password
export class PasswordChangeForm extends AccountForm {
	constructor() {
		super({
			tag: "form",
			needCaptcha: true,
		});
		this.renderPublicForm("/html/change-password").then(() =>
			validatePasswordMatch(this.el, "newPassword", "repeat"))
	}

	protected send() {
		this.postResponse("/api/change-password", req => {
			req["old"] = this.inputElement("oldPassword").value
			req["new"] = this.inputElement("newPassword").value
		})
	}
}
