import {inputValue} from '../util'
import {mod as lang, ui} from '../lang'
import {send, handlers, message} from '../connection'
import {responseCode} from './login'
import AccountFormView, {renderFields, validatePasswordMatch} from './common'
import {Captcha} from '../captcha'
import {read} from '../render'

interface PasswordChangeRequest extends Captcha {
	old: string
	new: string
}

// View for changing a password, that gets embedded below the parent view
export default class PasswordChangeView extends AccountFormView {
	constructor() {
		super({}, () =>
			this.sendRequest())
		this.render()
		read(() =>
			validatePasswordMatch(this.el, "newPassword", "repeat"))

		handlers[message.changePassword] = (msg: responseCode) =>
			this.handleResponse(msg)
	}

	// Render the element
	render() {
		const html = renderFields("oldPassword", "newPassword", "repeat")
		this.renderForm(html)
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

		this.reloadCaptcha(code)
		this.renderFormResponse(text)
	}

	// Also remove the websocket message handler, so this instance can be GCed
	remove() {
		delete handlers[message.changePassword]
		super.remove()
	}

	sendRequest() {
		const req: PasswordChangeRequest = {
			old: inputValue(this.el, "oldPassword"),
			new: inputValue(this.el, "newPassword"),
		} as PasswordChangeRequest
		this.injectCaptcha(req)
		send(message.changePassword, req)
	}
}
