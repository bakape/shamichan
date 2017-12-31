// File upload via drag and drop

import { postSM, postEvent } from "."
import { trigger } from "../../util"
import { page, boardConfig } from "../../state"
import FormModel from "./model"
import { expandThreadForm } from "./threads"

// Handle file drop
function onDrop(e: DragEvent) {
	const { files } = e.dataTransfer
	if (!files.length || isFileInput(e.target)) {
		return
	}

	e.stopPropagation()
	e.preventDefault()

	if (!page.thread) {
		expandThreadForm();
		(document
			.querySelector("#new-thread-form input[type=file]") as any)
			.files = files
		return
	}

	if (boardConfig.textOnly) {
		return
	}

	// Create form, if none
	postSM.feed(postEvent.open)

	// Neither disconnected, errored or already has image
	const m = trigger("getPostModel") as FormModel
	if (m && !m.image) {
		m.uploadFile(files)
	}
}

// Returns, if event target is an <input type=file>
function isFileInput(target: EventTarget): boolean {
	const el = target as HTMLElement
	return el.tagName === "INPUT" && el.getAttribute("type") === "file"
}

function stopDefault(e: Event) {
	if (!isFileInput(e.target)) {
		e.stopPropagation()
		e.preventDefault()
	}
}

// Bind listeners
export default () => {
	for (let event of ["dragenter", "dragexit", "dragover"]) {
		document.addEventListener(event, stopDefault)
	}
	document.addEventListener("drop", onDrop)
}
