import { SelectedBoardForm, newRequest } from "./common"

export default class BoardDeletionView extends SelectedBoardForm {
	constructor() {
		super({})
	}

	public renderNext(board: string) {
		this.board = board
		this.renderPublicForm("/forms/captcha")
	}

	protected send() {
		const req = newRequest()
		req["id"] = this.board
		this.injectCaptcha(req)
		this.postResponse("/admin/deleteBoard", req)
	}
}
