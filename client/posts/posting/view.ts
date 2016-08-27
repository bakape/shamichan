// View classes for post authoring

import PostView, {OPView} from "../view"
import {FormModel} from "./model"
import {Post, OP} from "../models"
import {isMobile} from "../../state"
import {setAttrs, makeFrag, applyMixins} from "../../util"
import {parseTerminatedLine} from "../render/body"
import {write} from "../../render"
import {ui} from "../../lang"

// Post creation and update view
export class FormView extends PostView {
	model: Post & FormModel
	inputLock: boolean
	$input: HTMLSpanElement
	$done: HTMLInputElement
	$postControls: Element

	constructor(model: Post) {
		super(model)
		this.renderInputs()
	}

	// Render extra input fields for inputting text and optionally uploading
	// images
	renderInputs() {
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

		this.$done = document.createElement("input")
		setAttrs(this.$done, {
			name: "done",
			type: "button",
			value: ui.done,
		})
		this.$done.addEventListener("click", () =>
			this.model.commitClose())
		this.$postControls.append(this.$done)

		write(() => {
			this.$blockquote.innerHTML = ""
			this.$blockquote.append(this.$input)
			this.el.append(this.$postControls)
			this.$input.focus()
		})

		// TODO: UploadForm integrations

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
}

// FormView of an OP post
export class OPFormView extends FormView implements OPView {
	$omit: Element
	model: OP & FormModel
	renderOmit: () => void

	constructor(model: OP) {
		super(model)
	}
}

applyMixins(OPFormView, OPView)