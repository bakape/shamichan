import AccountFormView, { newRequest, LoginCredentials } from './common'
import { makeFrag, inputValue } from '../util'
import { fetchHTML, postJSON } from "../fetch"

interface BoardCreationRequest extends LoginCredentials {
	name: string
	title: string
}

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
		const req = newRequest<BoardCreationRequest>()
		req.name = inputValue(this.el, 'boardName')
		req.title = inputValue(this.el, 'boardTitle')
		this.injectCaptcha(req)

		const res = await postJSON("/admin/createBoard", req)
		switch (res.status) {
			case 200:
				this.remove()
				break
			default:
				this.reloadCaptcha()
				this.renderFormResponse(await res.text())
		}
	}
}
