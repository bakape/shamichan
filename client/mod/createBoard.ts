import AccountFormView, { newRequest } from './common'
import { makeFrag, inputValue } from '../util'
import { fetchHTML } from "../fetch"

// Panel view for creating boards
export default class BoardCreationPanel extends AccountFormView {
	constructor() {
		super({ tag: "form" }, () =>
			this.onSubmit())
		this.render()
	}

	protected async render() {
		const [html, err] = await fetchHTML("/forms/createBoard")
		if (err) {
			throw err
		}
		this.el.append(makeFrag(html))
		super.render()
	}

	private async onSubmit() {
		const req = newRequest()
		req["name"] = inputValue(this.el, 'boardName')
		req["title"] = inputValue(this.el, 'boardTitle')
		this.injectCaptcha(req)

		this.postJSON("/admin/createBoard", req)
	}
}
