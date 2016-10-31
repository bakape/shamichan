// File upload via drag and drop

import {$threads} from "../../render"
import {postSM, postEvent, postModel} from "./main"
import {page, boardConfig} from "../../state"
import {ReplyFormModel} from "./model"

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
		(postModel as ReplyFormModel).uploadFile(files[0])
	}
}

function stopDefault(e: Event) {
	// No drag and drop for thread creation right now. Keep default behavior.
	if (page.thread) {
		e.stopPropagation()
		e.preventDefault()
	}
}

// Bind listeners
for (let event of ["dragenter", "dragexit", "dragover"]) {
	$threads.addEventListener(event, stopDefault)
}
$threads.addEventListener("drop", onDrop)
