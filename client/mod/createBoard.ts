import {FormView, renderInput, InputSpec, inputType} from './util'
import Model from '../model'
import AccountPanel from './login'
import {send, message, handlers} from '../connection'
import {inputValue, table} from '../util'
import {admin as lang, mod, fetchAdminPack} from '../lang'
import {write} from '../render'

// Response codes for board creation requests
const enum responseCode {
	boardCreated,
	boardNameTaken,
	boardNameTooLong,
	titleTooLong,
}

// Panel view for creating boards
export default class BoardCreationPanel extends FormView<Model> {
	constructor(parent: AccountPanel) {
		super({parent}, el =>
			send(message.createBoard, {
				name: inputValue(el, 'boardName'),
				title: inputValue(el, 'boardTitle'),
			}))
		fetchAdminPack().then(() =>
			this.render())
		handlers[message.createBoard] = (res: responseCode) =>
			this.handleResponse(res)
	}

	render() {
		const html = table(['boardName', 'boardTitle'], name => {
			const [label, tooltip] = lang[name]
			return renderInput({
				name,
				label,
				tooltip,
				type: inputType.string,
				minLength: 1,
				maxLength: name === "boardName" ? 3 : 100
			})
		})
		this.renderForm(html)
	}

	remove() {
		delete handlers[message.createBoard]
		super.remove()
	}

	handleResponse(res: responseCode) {
		let text = ""
		switch (res) {
		case responseCode.boardCreated:
			this.remove()
			return
		case responseCode.boardNameTaken:
			text = lang.boardNameTaken
			break
		default:
			text = mod.theFuck // Should not happen
		}
		write(() =>
			this.el
			.querySelector(".form-response")
			.textContent = text)
	}
}
