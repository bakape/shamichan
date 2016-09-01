import {
	on, inputValue, applyMixins, fetchBoardList, fetchBoarConfigs, makeFrag,
} from '../../util'
import {write, $threads} from '../../render'
import {FormView, inputType, renderInput, InputSpec} from '../../forms'
import {Captcha} from '../../captcha'
import {PostCredentials, newAllocRequest} from './identity'
import {page, boardConfig} from '../../state'
import {posts as lang, ui} from '../../lang'
import {send, message, handlers} from '../../connection'
import UploadForm, {FileData} from './upload'
import navigate from '../../history'
import {OPFormModel} from './model'

interface ThreadCreationRequest extends PostCredentials, Captcha {
	subject: string
	board: string
	image?: FileData
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
	$uploadContainer: HTMLSpanElement
	needImage: boolean = true // Does the board require an OP image?
	selectedBoard: string

	// UploadForm properties
	$spoiler: HTMLSpanElement
	$uploadStatus: Element
	$uploadInput: HTMLInputElement
	renderUploadForm: () => void
	uploadFile: () => Promise<FileData>
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
		const frag = document.createDocumentFragment()

		// Have the user to select the target board, if on the "/all/"
		// metaboard
		if (page.board === "all") {
			frag.append(await this.initBoardSelection())
		}

		const html = renderField({
			name: "subject",
			type: inputType.string,
			maxLength: 100,
			required: true,
		})
		frag.append(makeFrag(html))

		// Render image upload controls
		if (!boardConfig.textOnly) {
			this.renderUploadForm()
			this.$uploadContainer = document.createElement("span")
			this.$uploadContainer.setAttribute("class", "upload-container")
			this.$uploadContainer.append(
				this.$spoiler,
				this.$uploadStatus,
				document.createElement("br"),
				this.$uploadInput,
				document.createElement("br"),
			)
			if (!this.needImage) {
				this.$uploadContainer.style.display = "none"
			}
			frag.append(this.$uploadContainer)
		}

		this.renderForm(frag)
		write(() =>
			(this.$aside.classList.add("expanded"),
			this.$aside.append(this.el),
			(this.el.querySelector("input, select") as HTMLElement).focus()))
	}

	// Initialize the board selection input for the /all/ board and return its
	// HTML
	async initBoardSelection(): Promise<DocumentFragment> {
		// TODO: Some kind of more elegant selection panel

		// Hide the image upload controls, if the first board on the list
		// is a text-only board
		const boards = await fetchBoardList(),
			[first] = boards
		if (first && (await fetchBoarConfigs(first.id)).textOnly) {
			this.needImage = false
		}

		const html = renderField({
			name: "board",
			type: inputType.select,
			choices: boards.map(({title, id}) =>
				`${id} - ${title}`),
		})
		const frag = makeFrag(html)

		// Assign and bind event listener for changes to the board selection
		this.$board = frag
			.querySelector("select[name=board]") as HTMLSelectElement
		on(this.$board, "input", () =>
			this.toggleUploadForm())

		return frag
	}

	// When on the /all/ board, you may possibly post to boards that are
	// configured text-only. If a text-only board is selected, hide the upload
	// inputs.
	async toggleUploadForm() {
		const {textOnly} = await fetchBoarConfigs(this.getSelectedBoard()),
			display = textOnly ? "none" : ""
		this.needImage = !textOnly
		write(() => {
			(this.$uploadContainer as HTMLElement).style.display = display
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
		const req = newAllocRequest() as ThreadCreationRequest

		if (this.needImage) {
			req.image = await this.uploadFile()
			if (!req.image) {
				this.reloadCaptcha(1)
				return
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
