// View classes for post authoring

import PostView, { OPView } from "../view"
import { ReplyFormModel } from "./model"
import { Post, OP } from "../models"
import { isMobile, boardConfig } from "../../state"
import { setAttrs, makeFrag, applyMixins } from "../../util"
import { parseTerminatedLine } from "../render/body"
import { renderHeader, renderName } from "../render/posts"
import { write, $threads } from "../../render"
import { ui } from "../../lang"
import { $threadContainer } from "../../page/thread"
import { postSM, postEvent } from "./main"
import UploadForm, { FileData } from "./upload"
import identity from "./identity"

// Element at the bottom of the thread to keep the fixed reply form from
// overlaping any other posts, when scrolled till bottom
let $bottomSpacer: HTMLElement

// Post creation and update view
export class FormView extends PostView implements UploadForm {
	el: HTMLElement
	model: ReplyFormModel
	inputLock: boolean
	$input: HTMLSpanElement
	$done: HTMLInputElement
	$cancel: HTMLInputElement
	observer: MutationObserver
	$postControls: Element
	previousHeight: number

	// UploadForm properties
	$spoiler: HTMLSpanElement
	$uploadStatus: HTMLSpanElement
	$uploadInput: HTMLInputElement
	renderUploadForm: () => void
	uploadFile: (file?: File) => Promise<FileData>
	upload: (file: File) => Promise<string>
	renderProgress: (event: ProgressEvent) => void
	spoilerImage: () => Promise<void>

	[index: string]: any

	constructor(model: Post, isOP: boolean) {
		super(model)
		this.renderInputs(isOP)
		if (!isOP) {
			this.el.classList.add("reply-form")
			this.initDraft()
		}
	}

	// Render extra input fields for inputting text and optionally uploading
	// images
	renderInputs(isOP: boolean) {
		this.$input = document.createElement("span")
		const attrs: { [key: string]: string } = {
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
			.append(isOP ? this.renderDone() : this.renderDraft())

		write(() => {
			this.$blockquote.innerHTML = ""
			this.$blockquote.append(this.$input)
			this.el.querySelector(".post-container").append(this.$postControls)
			this.$input.focus()
		})
	}

	// Aditional controls and header contents for unallocated draft forms
	renderDraft(): DocumentFragment {
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

		this.renderIndentity()

		return frag
	}

	// Render a temporary view of the identity fields, so the user can see what
	// credentials he is about to post with
	renderIndentity() {
		let {name, email} = identity,
			trip = ""
		const iHash = name.indexOf("#")
		if (iHash !== -1) {
			trip = "?"
			name = name.slice(0, iHash)
		}
		renderName(this.el.querySelector(".name"), {
			trip,
			name: name.trim(),
			email: email.trim(),
			auth: undefined,
		})
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
		$bottomSpacer = document.getElementById("bottom-spacer")

		// Keep this post and $bottomSpacer the same height
		this.observer = new MutationObserver(() =>
			write(() =>
				this.resizeSpacer()))
		this.observer.observe(this.el, {
			childList: true,
			attributes: true,
			characterData: true,
			subtree: true,
		})

		write(() => {
			$threadContainer.append(this.el)
			this.$input.focus()
			this.resizeSpacer()
		})
	}

	// Resize $bottomSpacer to the same top position as this post
	resizeSpacer() {
		// Not a reply
		if (!$bottomSpacer) {
			return
		}

		// Avoid spacer being seen, if thread is too short to fill the
		// viewport.
		if ($threadContainer.offsetHeight < $threads.offsetHeight) {
			return
		}

		const {height} = this.el.getBoundingClientRect()
		// Avoid needless writes
		if (this.previousHeight === height) {
			return
		}
		this.previousHeight = height
		$bottomSpacer.style.height = `calc(${height}px - 2.1em)`
	}

	removeUploadForm() {
		write(() => {
			this.$uploadInput.remove()
			this.$uploadStatus.remove()
		})
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

	// Transform form into a generic post. Removes any dangling form controls
	// and frees up references.
	cleanUp() {
		write(() => {
			this.el.classList.remove("reply-form")
			if (this.$postControls) {
				this.$postControls.remove()
			}
			if ($bottomSpacer) {
				$bottomSpacer.style.height = ""
			}
			if (this.observer) {
				this.observer.disconnect()
			}
			this.$postControls
				= $bottomSpacer
				= this.observer
				= this.$done
				= this.$cancel
				= this.$input
				= this.$uploadInput
				= this.$uploadStatus
				= this.$spoiler
				= null
		})
	}

	// Clean up on form removal
	remove() {
		super.remove()
		this.cleanUp()
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
		write(() => {
			this.el.id = this.id as string
			$header.classList.remove("temporary")
			renderHeader($header, this.model)
			this.$cancel.remove()
			this.$postControls.prepend(this.renderDone())
		})
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

	// Insert image into the open post
	insertImage() {
		this.renderImage()
		this.removeUploadForm()

		const {$spoiler} = this
		if (this.model.image.spoiler) {
			write(() =>
				$spoiler.remove())
		} else {
			$spoiler.addEventListener("change", () => this.spoilerImage(), {
				passive: true,
			})
		}
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
