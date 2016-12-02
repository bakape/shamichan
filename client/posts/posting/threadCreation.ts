import { on, inputValue } from '../../util'
import { write, threads } from '../../render'
import FormView from '../../forms'
import { newAllocRequest } from './identity'
import { page, boardConfig } from '../../state'
import { send, message, handlers } from '../../connection'
import UploadForm from './upload'
import FormModel from "./model"
import navigate from '../../history'
import lang from "../../lang"

// Form view for creating new threads
class ThreadForm extends FormView {
	private aside: Element
	private selectedBoard: string
	private upload: UploadForm

	constructor(event: Event) {
		const aside = (event.target as Element).closest("aside"),
			el = aside.querySelector("#new-thread-form")
		super({ el }, () =>
			this.sendRequest())
		this.aside = aside
		this.render()
		handlers[message.postID] = (msg: number) =>
			this.handleResponse(msg)
	}

	// Render the element, hide the parent element's existing contents and
	// hide the "["..."]" encasing it
	private render() {
		if (!boardConfig.textOnly) {
			this.upload = new UploadForm(null, this.el)
		}
		write(() => {
			this.aside.classList.add("expanded")
			this.el.querySelector("input, select").focus()
		})
	}

	// Reset new thread form to initial state
	public remove() {
		delete handlers[message.postID]
		write(() => {
			for (let cl of ["expanded", "sending"]) {
				this.aside.classList.remove(cl)
			}
		})
	}

	private async sendRequest() {
		write(() => {
			this.el.querySelector("input[type=submit]").remove()
			this.el.querySelector("input[name=cancel]").remove()
			this.upload.el.querySelector("br:last-child").remove()
		})

		const req = newAllocRequest()

		if (this.upload && this.upload.input.files.length) {
			req["image"] = await this.upload.uploadFile()
			if (!req["image"]) {
				this.reloadCaptcha()
				return
			}
		}

		req["subject"] = inputValue(this.el, "subject")

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

	private async handleResponse(id: number) {
		if (id === -1) {
			this.renderFormResponse(lang.ui["invalidCaptcha"])
			this.reloadCaptcha()
			return
		}
		await navigate(`/${this.selectedBoard}/${id}`, null, true)
		new FormModel(id)
	}
}

on(threads, "click", e => new ThreadForm(e), {
	selector: ".new-thread-button",
	passive: true,
})
