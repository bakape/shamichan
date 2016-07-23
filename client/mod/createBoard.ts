import {FormView, renderInput, InputSpec, inputType} from './util'
import Model from '../model'
import AccountPanel, {renderFormResponse} from './login'
import {send, message, handlers} from '../connection'
import {inputValue, table} from '../util'
import {admin as lang, mod, fetchAdminPack, ui} from '../lang'
import {write} from '../render'
import {config} from '../state'

// Response codes for board creation requests
const enum responseCode {
	success,
	invalidBoardName,
	boardNameTaken,
	titleTooLong,
	invalidCaptcha,
}

// Panel view for creating boards
export default class BoardCreationPanel extends FormView<Model> {
	constructor(parent: AccountPanel) {
		super({parent, id: "create-board"}, el =>
			this.sendRequest(el))
		fetchAdminPack().then(() =>
			this.render())
		handlers[message.createBoard] = (res: responseCode) =>
			this.handleResponse(res)
	}

	render() {
		const html = table(['boardName', 'boardTitle'], name => {
			const [label, tooltip] = lang[name]
			const spec: InputSpec = {
				name,
				label,
				tooltip,
				type: inputType.string,
				minLength: 1,
			}
			if (name === "boardName") {
				spec.maxLength = 3
				spec.pattern = "^[a-z0-9]{1,3}$"
			} else {
				spec.maxLength = 100
			}

			return renderInput(spec)
		})

		this.renderForm(html)
	}

	remove() {
		delete handlers[message.createBoard]
		super.remove()
	}

	sendRequest(el: Element) {
		const req = {
			name: inputValue(el, 'boardName'),
			title: inputValue(el, 'boardTitle'),
		}
		this.injectCaptcha(req)
		send(message.createBoard, req)
	}

	handleResponse(res: responseCode) {
		let text: string
		switch (res) {
		case responseCode.success:
			this.remove()
			return
		case responseCode.boardNameTaken:
			text = lang.boardNameTaken
			break
		case responseCode.invalidCaptcha:
			text = ui.invalidCaptcha
			break
		default:
			text = mod.theFuck // Should not happen
		}

		this.reloadCaptcha(res)
		renderFormResponse(this.el, text)
	}
}
