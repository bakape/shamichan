// File upload via drag and drop

import { postSM, postEvent, postModel } from "."
import { page, boardConfig } from "../../state"
import FormModel from "./model"

// Handle file drop
function onDrop(e: DragEvent) {
	const {files} = e.dataTransfer

	// TODO: Drag & drop for thread creation
	if (!files.length || !page.thread) {
		return
	}

	e.stopPropagation()
	e.preventDefault()

	if (boardConfig.textOnly) {
		return
	}

	// Create form, if none
	postSM.feed(postEvent.open)

	// Neither disconnected, errored or already has image
	if (postModel && !postModel.image) {
		(postModel as FormModel).uploadFile(files[0])
	}
}

function stopDefault(e: Event) {
	// No drag and drop for thread creation right now. Keep default behavior.
	if (page.thread) {
		e.stopPropagation()
		e.preventDefault()
	}
}

export default () => {
	// Bind listeners
	const threads = document.getElementById("threads")
	for (let event of ["dragenter", "dragexit", "dragover"]) {
		threads.addEventListener(event, stopDefault)
	}
	threads.addEventListener("drop", onDrop)
}
