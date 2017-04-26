import { on } from '../../util'
import { FormView, navigate } from '../../ui'
import { newAllocRequest } from './identity'
import { page, boardConfig } from '../../state'
import { send, message, handlers } from '../../connection'
import UploadForm from './upload'
import FormModel from "./model"
import lang from "../../lang"

// Form view for creating new threads
class ThreadForm extends FormView {
	private aside: Element
	private selectedBoard: string
	private upload: UploadForm
	private submitEl: HTMLElement
	private lastBr: HTMLElement

	constructor(event: Event) {
		const aside = (event.target as Element).closest("aside")
		super({ el: document.getElementById("new-thread-form") })
		this.aside = aside
		this.render()
		this.submitEl = this.el.querySelector("input[type=submit]")
		handlers[message.postID] = (msg: number) =>
			this.handleResponse(msg)
	}

	// Render the element, hide the parent element's existing contents and
	// hide the "["..."]" encasing it
	private render() {
		if (!boardConfig.textOnly) {
			this.upload = new UploadForm(null, this.el)
			this.lastBr = this.upload.el.querySelector("br:last-child")
		}
		this.aside.classList.add("expanded")
		this.el.querySelector("input, select").focus()
	}

	// Reset new thread form to initial state and cancel upload
	public remove() {
		delete handlers[message.postID]
		if (this.upload && this.upload.isUploading) {
			this.upload.cancel()
		}
		this.reset()
		this.aside.classList.remove("expanded")
	}

	protected async send() {
		const req = newAllocRequest()
		this.submitEl.style.display = "none"

		if (this.upload && this.upload.input.files.length) {
			this.lastBr.style.display = "none"
			req["image"] = await this.upload.uploadFile()
			if (!req["image"]) {
				return this.reset()
			}
		}

		req["subject"] = this.inputElement("subject").value

		let board = page.board
		if (board === "all") {
			board = (this.el
				.querySelector("select[name=board]") as HTMLInputElement)
				.value
		}
		this.selectedBoard = req["board"] = board

		this.injectCaptcha(req)
		send(message.insertThread, req)
	}

	private reset() {
		this.reloadCaptcha()
		this.submitEl.style.display = ""
		if (this.upload) {
			this.lastBr.style.display = ""
		}
	}

	private async handleResponse(id: number) {
		if (id === -1) {
			this.renderFormResponse(lang.ui["invalidCaptcha"])
			this.reset()
			return
		}
		await navigate(`/${this.selectedBoard}/${id}`, null, true)
		new FormModel(id)
	}
}

export default () =>
	on(document.getElementById("threads"), "click", e => new ThreadForm(e), {
		selector: ".new-thread-button",
		passive: true,
	})
