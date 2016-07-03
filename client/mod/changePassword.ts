import {table, makeAttrs, inputValue} from '../util'
import {mod as lang} from '../lang'
import {write} from '../render'
import {send, handlers, message} from '../connection'
import AccountPanel, {validatePasswordMatch} from './login'
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

		handlers[message.changePassword] = (msg: boolean) =>
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
	handleResponse(success: boolean) {
		write(() => {
			if (success) {
				this.remove()
			} else {
				this.el
					.querySelector(".form-response")
					.textContent = lang.wrongPassword
			}
		})
	}

	// Also remove the websocket message handler, so this instance can be GCed
	remove() {
		delete handlers[message.changePassword]
		super.remove()
	}
}
