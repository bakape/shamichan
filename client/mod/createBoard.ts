import { AccountFormView, newRequest } from './common'
import { inputValue } from '../util'

// Panel view for creating boards
export default class BoardCreationPanel extends AccountFormView {
	constructor() {
		super({ tag: "form" })
		this.renderPublicForm("/forms/createBoard")
	}

	protected send() {
		const req = newRequest()
		req["name"] = inputValue(this.el, 'boardName')
		req["title"] = inputValue(this.el, 'boardTitle')
		this.injectCaptcha(req)

		this.postResponse("/admin/createBoard", req)
	}
}
