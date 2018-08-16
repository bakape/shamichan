// File upload via clipboard paste

import { postSM, postEvent } from "."
import { trigger } from "../../util"
import { page, boardConfig } from "../../state"
import FormModel from "./model"
import { expandThreadForm } from "./threads"

// Handle file paste
function onPaste(e: ClipboardEvent) {
	const files = e.clipboardData.files

	if (!(files.length == 1)) {
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

	// Neither disconnected, erred or already has image
	const m = trigger("getPostModel") as FormModel

	if (m && !m.image) {
		m.uploadFile(files)
	}
}

// Bind listeners
export default () => {
	document.addEventListener("paste", onPaste)
}
