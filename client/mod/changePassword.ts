import {table, makeAttrs, inputValue} from '../util'
import {mod as lang, ui} from '../lang'
import {write} from '../render'
import {send, handlers, message} from '../connection'
import AccountPanel, {
	validatePasswordMatch, responseCode, renderFormResponse
} from './login'
import {FormView} from './util'
import Model from '../model'

type PasswordChangeRequest = {
	old: string
	new: string
}

// View for changing a password, that gets embedded below the parent view
export default class PasswordChangeView extends FormView<Model> {
	constructor(parent: AccountPanel) {
		super({parent}, el =>
			send(message.changePassword, {
				old: inputValue(el, "oldPassword"),
				new: inputValue(el, "newPassword"),
			}))
		this.render()

		validatePasswordMatch(this.el, "newPassword", "repeat")

		handlers[message.changePassword] = (msg: responseCode) =>
			this.handleResponse(msg)
	}

	// Render the element
	render() {
		const tableData = ["oldPassword", "newPassword", "repeat"]
		const tableHTML = table(tableData, name => {
			const attrs: StringMap = {
				name,
				type: "password",
				minlength: "6",
				maxlength: "30",
				required: "",
			}
			return [
				`<label for="${name}">${lang[name]}:</label>`,
				`<input ${makeAttrs(attrs)}>`
			]
		})

		this.renderForm(tableHTML)
	}

	// Handle the changePassword response from the server
	handleResponse(code: responseCode) {
		let text: string
		switch (code) {
		case responseCode.success:
			this.remove()
			return
		case responseCode.wrongCredentials:
			text = lang.wrongPassword
			break
		case responseCode.invalidCaptcha:
			text = ui.invalidCaptcha
			break
		default:
			// Not supposed to happen, because of client-side form validation
			text = lang.theFuck
		}

		renderFormResponse(this.el, text)
	}

	// Also remove the websocket message handler, so this instance can be GCed
	remove() {
		delete handlers[message.changePassword]
		super.remove()
	}
}
