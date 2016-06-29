import View from '../view'
import {HTML, table, makeAttrs, inputValue} from '../util'
import {mod as lang, ui} from '../lang'
import {write, read} from '../render'
import {send, handlers, message} from '../connection'
import {validatePasswordMatch} from './login'

type PasswordChangeRequest = {
	old: string
	new: string
}

// View for changing a password, that gets embedded below the parent view
export default class PasswordChangeView extends View {
	$parent: Element

	constructor(parent: Element) {
		super({})
		this.$parent = parent
		this.render()
		write(() =>
			this.$parent.after(this.el))

		this.onClick({
			"input[name=cancel]": () =>
				this.remove()
		})
		this.on("submit", e =>
			this.submit(e))

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

		this.el.innerHTML = HTML
			`<form>
				${tableHTML}
				<input type="submit" value="${lang.submit}">
				<input type="button" name="cancel" value="${ui.cancel}">
			</form>
			<div class="form-response admin"></div>`
	}

	// Submit password change to server
	submit(event: Event) {
		event.preventDefault()
		const el = event.target as Element
		send(message.changePassword, {
			old: inputValue(el, "oldPassword"),
			new: inputValue(el, "newPassword"),
		})
	}

	// Handle the changePassword response from the server
	handleResponse(success: boolean) {
		if (success) {
			this.remove()
		} else {
			this.el
				.querySelector(".form-response")
				.textContent = lang.wrongPassword
		}
	}

	// Also remove the websocket message handler, so this instance can be GCed
	remove() {
		delete handlers[message.changePassword]
		super.remove()
	}
}
