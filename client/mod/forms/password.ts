import { AccountForm } from "./common"
import { validatePasswordMatch } from "../common"
import { inputValue } from "../../util"

// View for changing a password
export class PasswordChangeForm extends AccountForm {
	constructor() {
		super({ tag: "form" })
		this.renderPublicForm("/forms/changePassword").then(() =>
			validatePasswordMatch(this.el, "newPassword", "repeat"))
	}

	protected send() {
		this.postResponse("/admin/changePassword", req => {
			req["old"] = inputValue(this.el, "oldPassword")
			req["new"] = inputValue(this.el, "newPassword")
		})
	}
}
