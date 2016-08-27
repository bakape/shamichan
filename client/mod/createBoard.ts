import {renderInput, InputSpec, inputType} from '../forms'
import AccountFormView from './common'
import {send, message, handlers} from '../connection'
import {inputValue, table, makeFrag} from '../util'
import {admin as lang, mod, fetchAdminPack, ui} from '../lang'

// Response codes for board creation requests
const enum responseCode {
	success,
	invalidBoardName,
	boardNameTaken,
	titleTooLong,
	invalidCaptcha,
}

// Panel view for creating boards
export default class BoardCreationPanel extends AccountFormView {
	constructor() {
		super({}, () =>
			this.sendRequest())
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

		this.renderForm(makeFrag(html))
	}

	remove() {
		delete handlers[message.createBoard]
		super.remove()
	}

	sendRequest() {
		const req = {
			name: inputValue(this.el, 'boardName'),
			title: inputValue(this.el, 'boardTitle'),
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
		this.renderFormResponse(text)
	}
}
