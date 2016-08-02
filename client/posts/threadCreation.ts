import {
	HTML, on, inputValue, applyMixins, fetchBoardList, fetchBoarConfigs,
} from '../util'
import {$threads} from '../page/common'
import {write, read} from '../render'
import {FormView, inputType, renderInput, InputSpec} from '../forms'
import {Captcha} from '../captcha'
import identity from './identity'
import {page, boardConfig} from '../state'
import {posts as lang, ui} from '../lang'
import {send, message, handlers} from '../connection'
import UploadForm, {FileData} from './upload'

export interface PostCredentials extends Captcha, FileData {
	name?: string
	email?: string
	auth?: string // TODO
	password: string
}

interface ThreadCreationRequest extends PostCredentials {
	subject?: string
	board: string
	body: string
}

// Response codes for thread and post insertion requests
export const enum responseCode {success, invalidCaptcha}

// For ensuring we have unique captcha IDs
let threadFormCounter = 0

// Bind event listener to the thread container
export default () =>
	on($threads, "click", e => new ThreadForm(e),{
		selector: ".new-thread-button",
	})

// Form view for creating new threads
class ThreadForm extends FormView implements UploadForm {
	$aside: Element
	$board: Element
	$uploadContainer: Element
	needImage: boolean // Does the board require an OP image?

	// UploadForm properties
	$uploadStatus: Element
	$uploadInput: HTMLInputElement
	renderUploadForm: () => string
	uploadFile: (req: FileData) => Promise<boolean>
	renderProgress: (event: ProgressEvent) => void

	constructor(event: Event) {
		super({class: "new-thread-form"}, () =>
			this.sendRequest())
		this.$aside = (event.target as Element).closest("aside")
		this.render()

		handlers[message.insertThread] = (code: responseCode) =>
			this.handleResponse(code)
	}

	// Render the element, hide the parent element's existing contents and
	// hide the "["..."]" encasing it
	async render() {
		const specs: InputSpec[] = [
			{
				name: "subject",
				type: inputType.string,
				maxLength: 50,
			},
			{
				name: "body",
				type: inputType.multiline,
				required: true,
				rows: 4,
				maxLength: 2000,
			},
		]

		// Have the user to select the target board, if on the "/all/" metaboard
		if (page.board === "all") {

			// TODO: Some kind of more elegant selection panel

			// Hide the image upload controls, if the first board on the list
			// is a text-only board
			const boards = await fetchBoardList(),
				[first] = boards
			let display = ""
			this.needImage = true
			if (first && (await fetchBoarConfigs(first.id)).textOnly) {
				display = "none"
				this.needImage = false
			}
			read(() => {
				// Bind event listener for changes to the board selection
				this.$board = this.el.querySelector("select[name=board]")
				on(this.$board, "input", () =>
					this.toggleUploadForm())

				this.$uploadContainer =
					this.el
					.querySelector(".upload-container")
				write(() =>
					this.$uploadContainer.style.display = display)
			})

			specs.unshift({
				name: "board",
				type: inputType.select,
				choices: boards.map(({title, id}) =>
					`${id} - ${title}`),
			})
		}

		let html = ""
		for (let spec of specs) {
			spec.label = lang[spec.name]
			spec.placeholders = true
			html += renderInput(spec)[1] + "<br>"
		}
		if (!boardConfig.textOnly) {
			html += this.renderUploadForm() + "<br>"
			if (page.board !== "all") {
				this.needImage = true
			}
		}

		this.renderForm(html)
		write(() => {
			const cls = this.$aside.classList
			cls.remove("act")
			cls.add("expanded")
			this.$aside.append(this.el)
		})
	}

	// When on the /all/ board, you may possibly post to boards that are
	// configured text-only. If a text-only board is selected, hide the upload
	// inputs.
	async toggleUploadForm() {
		const {textOnly} = await fetchBoarConfigs(this.getSelectedBoard()),
			display = textOnly ? "none" : ""
		this.needImage = !textOnly
		write(() => {
			this.$uploadContainer.style.display = display
			this.$uploadInput.disabled = textOnly
		})
	}

	// Retrieve the curently selected board, if on the /all/ board
	getSelectedBoard(): string {
		return this.$board.value.match(/^(\w+) -/)[1]
	}

	// Reset new thread form to initial state
	remove() {
		super.remove()
		write(() => {
			const cls = this.$aside.classList
			cls.remove("expanded")
			cls.add("act")
		})
	}

	async sendRequest() {
		const req: any = {
			password: identity.postPassword,
		} as ThreadCreationRequest

		if (this.needImage) {
			if (!(await this.uploadFile(req))) {
				this.reloadCaptcha(1)
				return
			}
		}

		for (let key of ["name", "email"]) {
			const val = identity[key]
			if (val) {
				req[key] = val
			}
		}
		const subject = inputValue(this.el, "subject")
		if (subject) {
			req.subject = subject
		}
		req.body = this.el.querySelector("textarea[name=body]").value
		req.board = page.board === "all" ? this.getSelectedBoard() : page.board
		this.injectCaptcha(req)
		send(message.insertThread, req)
	}

	handleResponse(code: responseCode) {
		switch (code) {
		case responseCode.success:
			this.remove()

			// TODO: Redirect to newly-created thread

			break
		case responseCode.invalidCaptcha:
			this.renderFormResponse(ui.invalidCaptcha)
			this.reloadCaptcha(code)
			break
		}
	}
}

applyMixins(ThreadForm, UploadForm)
