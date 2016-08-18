import {
	on, inputValue, applyMixins, fetchBoardList, fetchBoarConfigs,
} from '../../util'
import {write, read, $threads} from '../../render'
import {FormView, inputType, renderInput, InputSpec} from '../../forms'
import {Captcha} from '../../captcha'
import identity from './identity'
import {page, boardConfig} from '../../state'
import {posts as lang, ui} from '../../lang'
import {send, message, handlers} from '../../connection'
import UploadForm, {FileData} from './upload'
import navigate from '../../history'
import {OPFormModel} from './form'

export interface PostCredentials extends Captcha, FileData {
	name?: string
	email?: string
	auth?: string // TODO
	password: string
}

interface ThreadCreationRequest extends PostCredentials {
	subject: string
	board: string
}

// Response codes for thread and post insertion requests
export const enum responseCode {success, invalidCaptcha}

type ThreadCreationResponse = {
	code: responseCode
	id: number
}

// Bind event listener to the thread container
export default () =>
	on($threads, "click", e => new ThreadForm(e), {
		selector: ".new-thread-button",
	})

// Form view for creating new threads
class ThreadForm extends FormView implements UploadForm {
	$aside: Element
	$board: HTMLSelectElement
	$uploadContainer: Element
	needImage: boolean // Does the board require an OP image?
	selectedBoard: string

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
		handlers[message.insertThread] = (msg: ThreadCreationResponse) =>
			this.handleResponse(msg)
	}

	// Render the element, hide the parent element's existing contents and
	// hide the "["..."]" encasing it
	async render() {
		let html = ""

		// Have the user to select the target board, if on the "/all/" metaboard
		if (page.board === "all") {
			html += await this.initBoardSelection()
		}

		html += renderField({
			name: "subject",
			type: inputType.string,
			maxLength: 100,
			required: true,
		})
		if (!boardConfig.textOnly) {
			html += this.renderUploadForm() + "<br>"
			if (page.board !== "all") {
				this.needImage = true
			}
		}

		this.renderForm(html)
		write(() => {
			this.$aside.classList.add("expanded")
			this.$aside.append(this.el)
			this.el.querySelector("input, select").focus()
		})
	}

	// Initialize the board selection input for the /all/ board and return its
	// HTML
	async initBoardSelection(): Promise<string> {
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
			this.$board =
				this.el
				.querySelector("select[name=board]") as HTMLSelectElement
			on(this.$board, "input", () =>
				this.toggleUploadForm())

			this.$uploadContainer = this.el.querySelector(".upload-container")
			write(() =>
				this.$uploadContainer.style.display = display)
		})

		return renderField({
			name: "board",
			type: inputType.select,
			choices: boards.map(({title, id}) =>
				`${id} - ${title}`),
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
		write(() =>
			this.$aside.classList.remove("expanded"))
	}

	async sendRequest() {
		const req: ThreadCreationRequest = {
			password: identity.postPassword,
		} as any

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
		req.subject = inputValue(this.el, "subject")
		this.selectedBoard = req.board =
			page.board === "all"
				? this.getSelectedBoard()
				: page.board
		this.injectCaptcha(req)
		send(message.insertThread, req)
	}

	async handleResponse({code, id}: ThreadCreationResponse) {
		switch (code) {
		case responseCode.success:
			await navigate(`/${this.selectedBoard}/${id}`, null, true)
			new OPFormModel(id)
			break
		case responseCode.invalidCaptcha:
			this.renderFormResponse(ui.invalidCaptcha)
			this.reloadCaptcha(code)
			break
		}
	}
}

applyMixins(ThreadForm, UploadForm)

// Render a single field of the form
function renderField(spec: InputSpec): string {
	spec.label = lang[spec.name]
	spec.placeholders = true
	return renderInput(spec)[1] + "<br>"
}
