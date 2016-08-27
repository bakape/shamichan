// View classes for post authoring

import PostView, {OPView} from "../view"
import {FormModel} from "./model"
import {Post, OP} from "../models"
import {isMobile} from "../../state"
import {setAttrs, makeFrag, applyMixins} from "../../util"
import {parseTerminatedLine} from "../render/body"
import {write} from "../../render"
import {ui} from "../../lang"
import {$threadContainer} from "../../page/thread"
import {postSM, postEvent} from "./main"
import UploadForm, {FileData} from "./upload"

// Post creation and update view
export class FormView extends PostView implements UploadForm {
	model: Post & FormModel
	inputLock: boolean
	$input: HTMLSpanElement
	$done: HTMLInputElement
	$cancel: HTMLInputElement
	$postControls: Element

	// UploadForm properties
	$uploadStatus: Element
	$uploadInput: HTMLInputElement
	renderUploadForm: () => string
	uploadFile: () => Promise<FileData>
	renderProgress: (event: ProgressEvent) => void

	[index: string]: any

	constructor(model: Post, allocated: boolean) {
		super(model)
		this.renderInputs(allocated)
		if (!allocated) {
			this.initDraft()
		}
	}

	// Render extra input fields for inputting text and optionally uploading
	// images
	renderInputs(allocated: boolean) {
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
		this.$input.textContent = ""
		this.$input.addEventListener("input", (event: Event) =>
			this.onInput((event.target as Element).textContent))
		this.$input.addEventListener("keydown", (event: KeyboardEvent) =>
			this.onKeyDown(event))

		this.$postControls = document.createElement("div")
		this.$postControls.id = "post-controls"
		this.$postControls
			.append(allocated ? this.renderDone() : this.renderDraftInputs())

		write(() => {
			this.$blockquote.innerHTML = ""
			this.$blockquote.append(this.$input)
			this.el.append(this.$postControls)
			this.$input.focus()
		})
	}

	// Aditional controls for draft forms
	renderDraftInputs(): DocumentFragment {
		const frag = document.createDocumentFragment()
		const $cancel = this.createButton("cancel", () =>
			(postSM.feed(postEvent.done),
			this.remove()))
		frag.append($cancel)

		// TODO: UploadForm integrations

		return frag
	}

	// Button for closing allocated posts
	renderDone(): HTMLInputElement {
		return this.createButton("done", () =>
			this.model.commitClose())
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
			$threadContainer.append(this.el))
	}

	// Handle input events on $input
	onInput(val: string) {
		if (!this.inputLock) {
			this.model.parseInput(val)
		}
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
		const val = this.$input.textContent.slice(0, -length)
		write(() =>
			this.lockInput(() =>
				this.$input.textContent = val))
	}

	// Start a new line in the input field and close the previous one
	startNewLine() {
		const line = this.model.inputState.line.slice(0, -1),
			frag = makeFrag(parseTerminatedLine(line, this.model))
		write(() => {
			this.$input.before(frag)
			this.lockInput(() =>
				this.$input.textContent = "")
		})
	}

	// Inject lines before $input and set $input contents to lastLine
	injectLines(lines: string[], lastLine: string) {
		const frag = document.createDocumentFragment()
		for (let line of lines) {
			const el = makeFrag(parseTerminatedLine(line, this.model))
			frag.append(el)
		}
		write(() =>
			(this.$input.before(frag),
			this.lockInput(() =>
				this.$input.textContent = lastLine)))
	}

	// Parse and replace the temporary like closed by $input with a proper
	// parsed line
	terminateLine(num: number) {
		const html = parseTerminatedLine(this.model.lastBodyLine(), this.model),
			frag = makeFrag(html)
		write(() =>
			this.$blockquote.children[num].replaceWith(frag))
	}

	// Remove any dangling form controls deallocate references
	cleanUp() {
		write(() =>
			(this.$postControls.remove(),
			this.$postControls = this.$done = null))
	}

	// Lock the post form after a crytical error accours
	renderError() {
		write(() =>
			(this.$blockquote.classList.add("errored"),
			this.$input.setAttribute("contenteditable", "false")))
	}
}

// FormView of an OP post
export class OPFormView extends FormView implements OPView {
	$omit: Element
	model: OP & FormModel
	renderOmit: () => void

	constructor(model: OP) {
		super(model, true)
	}
}

applyMixins(OPFormView, OPView)
