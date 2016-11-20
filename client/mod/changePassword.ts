import { inputValue } from '../util'
import AccountFormView, { validatePasswordMatch, newRequest } from './common'

// View for changing a password
export default class PasswordChangeView extends AccountFormView {
	constructor() {
		super({ tag: "form" }, () =>
			this.sendRequest())
		this.renderPublicForm("/forms/changePassword").then(() =>
			validatePasswordMatch(this.el, "newPassword", "repeat"))
	}

	private async sendRequest() {
		const req = newRequest()
		req["old"] = inputValue(this.el, "oldPassword")
		req["new"] = inputValue(this.el, "newPassword")
		this.injectCaptcha(req)
		this.postResponse("/admin/changePassword", req)
	}
}
