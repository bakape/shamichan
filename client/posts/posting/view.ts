// View classes for post authoring

import PostView, {OPView} from "../view"
import {ReplyFormModel} from "./model"
import {Post, OP} from "../models"
import {isMobile, boardConfig} from "../../state"
import {setAttrs, makeFrag, applyMixins} from "../../util"
import {parseTerminatedLine} from "../render/body"
import {renderHeader} from "../render/posts"
import {write} from "../../render"
import {ui} from "../../lang"
import {$threadContainer} from "../../page/thread"
import {postSM, postEvent} from "./main"
import UploadForm, {FileData} from "./upload"

// Post creation and update view
export class FormView extends PostView implements UploadForm {
	model: ReplyFormModel
	inputLock: boolean
	$input: HTMLSpanElement
	$done: HTMLInputElement
	$cancel: HTMLInputElement
	$postControls: Element

	// UploadForm properties
	$spoiler: HTMLSpanElement
	$uploadStatus: HTMLSpanElement
	$uploadInput: HTMLInputElement
	renderUploadForm: () => void
	uploadFile: (file?: File) => Promise<FileData>
	renderProgress: (event: ProgressEvent) => void

	[index: string]: any

	constructor(model: Post, isOP: boolean) {
		super(model)
		this.renderInputs(isOP)
		if (!isOP) {
			this.initDraft()
		}
	}

	// Render extra input fields for inputting text and optionally uploading
	// images
	renderInputs(isOP: boolean) {
		this.$input = document.createElement("span")
		const attrs: {[key: string]: string} = {
			id: "text-input",
			name: "body",
			contenteditable: "",
		}
		if (isMobile) {
			attrs["autocomplete"] = ""
		}
		setAttrs(this.$input, attrs)

		// Always make sure the input span alwas has at least 1 character, so
		// it does not float onto the image, if any.
		this.$input.textContent = "\u200b"
		this.$input.addEventListener("input", (event: Event) => {
			event.stopImmediatePropagation()
			this.onInput((event.target as Element).textContent)
		})
		this.$input.addEventListener("keydown", (event: KeyboardEvent) =>
			this.onKeyDown(event))

		this.$postControls = document.createElement("div")
		this.$postControls.id = "post-controls"
		this.$postControls
			.append(isOP ? this.renderDone() : this.renderDraftInputs())

		write(() => {
			this.$blockquote.innerHTML = ""
			this.$blockquote.append(this.$input)
			this.el.querySelector(".post-container").append(this.$postControls)
			this.$input.focus()
		})
	}

	// Aditional controls for draft forms
	renderDraftInputs(): DocumentFragment {
		const frag = document.createDocumentFragment()
		const $cancel = this.createButton(
			"cancel",
			postSM.feeder(postEvent.done),
		)
		frag.append($cancel)

		if (!boardConfig.textOnly) {
			this.renderUploadForm()
			frag.append(this.$uploadInput, this.$spoiler, this.$uploadStatus)
			this.$uploadInput.addEventListener("change", () =>
				this.model.uploadFile())
		}

		return frag
	}

	// Button for closing allocated posts
	renderDone(): HTMLInputElement {
		return this.createButton("done", postSM.feeder(postEvent.done))
	}

	// Create a clickable button element
	createButton(name: string, clickHandler: () => void): HTMLInputElement {
		const el = document.createElement("input")
		setAttrs(el, {
			name,
			type: "button",
			value: ui[name],
		})
		el.addEventListener("click", clickHandler)
		return this["$" + name] = el
	}

	// Initialize extra elements for a draft unallocated post
	initDraft() {
		this.el.querySelector("header").classList.add("temporary")
		write(() =>
			($threadContainer.append(this.el),
			this.$input.focus()))
	}

	removeUploadForm() {
		write(() =>
			(this.$uploadInput.remove(),
			this.$uploadStatus.remove()))
	}

	// Handle input events on $input
	onInput(val: string = this.$input.textContent) {
		if (this.inputLock) {
			return
		}
		if (val === "") {
			this.lockInput(() =>
				this.$input.textContent = "\u200b")
		}
		this.model.parseInput(val.replace("\u200b", ""))
	}

	// Ignore any oninput events on $input during suplied function call
	lockInput(fn: () => void) {
		this.inputLock = true
		fn()
		this.inputLock = false
	}

	// Handle keydown events on $input
	onKeyDown(event: KeyboardEvent) {
		if (event.which === 13) { // Enter
			event.preventDefault()
			return this.onInput(this.model.inputState.line + "\n")
		}
	}

	// Trim $input from the end by the suplied length
	trimInput(length: number) {
		let val = this.$input.textContent.slice(0, -length) || "\u200b"
		write(() =>
			this.lockInput(() =>
				this.$input.textContent = val))
	}


	// Replace the current line and set the cursor to the input's end
	replaceLine(line: string) {
		write(() => {
			this.$input.textContent = line || "\u200b"
			const range = document.createRange(),
				sel = window.getSelection()
			range.setEndAfter(this.$input.lastChild)
			range.collapse(false)
			sel.removeAllRanges()
			sel.addRange(range)
			this.onInput()
		})
	}

	// Start a new line in the input field and close the previous one
	startNewLine() {
		const {line} = this.model.inputState,
			frag = makeFrag(parseTerminatedLine(line, this.model))
		write(() => {
			this.$input.before(frag)
			this.lockInput(() =>
				this.$input.textContent = "\u200b")
		})
	}

	// Inject lines before $input and set $input contents to the lastLine
	injectLines(lines: string[], lastLine: string) {
		const frag = document.createDocumentFragment()
		for (let line of lines) {
			const el = makeFrag(parseTerminatedLine(line, this.model))
			frag.append(el)
		}
		write(() =>
			this.$input.before(frag))
		this.replaceLine(lastLine)
	}

	// Parse and replace the temporary like closed by $input with a proper
	// parsed line
	terminateLine(num: number) {
		const html = parseTerminatedLine(this.model.lastBodyLine(), this.model),
			frag = makeFrag(html)
		write(() =>
			this.$blockquote.children[num].replaceWith(frag))
	}

	// Remove any dangling form controls to deallocate referenced elements
	cleanUp() {
		write(() =>
			(this.$postControls.remove(),
			this.$postControls
				= this.$done
				= this.$cancel
				= this.$input
				= this.$uploadInput
				= this.$uploadStatus
				= this.$spoiler
				= null))
	}

	// Lock the post form after a crytical error accours
	renderError() {
		write(() =>
			(this.el.classList.add("errored"),
			this.$input.setAttribute("contenteditable", "false")))
	}

	// Transition into allocated post
	renderAlloc() {
		this.id = "p" + this.model.id
		const $header = this.el.querySelector("header")
		write(() =>
			(this.el.id = this.id as string,
			$header.classList.remove("temporary"),
			renderHeader($header, this.model),
			this.$cancel.remove(),
			this.$postControls.prepend(this.renderDone())))
	}

	// Toggle the spoiler input checkbox
	toggleSpoiler() {
		// Can only turn a spoiler on, if image already allocated
		if (this.model.image && this.model.image.spoiler) {
			return
		}

		write(() => {
			const el = this.el
				.querySelector("input[name=spoiler]") as HTMLInputElement
			el.checked = !el.checked
		})
	}
}

applyMixins(FormView, UploadForm)

// FormView of an OP post
export class OPFormView extends FormView implements OPView {
	$omit: Element
	model: any
	renderOmit: () => void

	constructor(model: OP) {
		super(model, true)
	}
}

applyMixins(OPFormView, OPView)
