// File upload via drag and drop

import { postSM, postEvent } from "."
import { trigger } from "../../util"
import { page, boardConfig } from "../../state"
import FormModel from "./model"
import { expandThreadForm } from "./threads"

// Handle file drop
function onDrop(e: DragEvent) {
	const { files } = e.dataTransfer
	const target = e.target as HTMLElement

	if (!files.length
		|| (target.matches && target.matches("input[type=file]"))
	) {
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

function stopDefault(e: Event) {
	e.stopPropagation()
	e.preventDefault()
}

// Bind listeners
export default () => {
	for (let event of ["dragenter", "dragexit", "dragover"]) {
		document.addEventListener(event, stopDefault)
	}
	document.addEventListener("drop", onDrop)
}
