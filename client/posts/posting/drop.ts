// File upload via drag and drop

import { postSM, postEvent } from "."
import { trigger } from "../../util"
import { page, boardConfig } from "../../state"
import FormModel from "./model"
import { expandThreadForm } from "./threads"

// Handle file drop
async function onDrop(e: DragEvent) {
	const { files } = e.dataTransfer;
	const url = e.dataTransfer.getData("text/uri-list");
	if ((!files.length && !url) || isFileInput(e.target)) {
		return
	}

	e.stopPropagation()
	e.preventDefault()

	// Fetch file from link
	let file: File;
	if (!files.length) {
		try {
			file = new File([await (await fetch(url)).blob()], "download");
		} catch (err) {
			alert(err);
			return;
		}
	}

	if (!page.thread) {
		if (!files.length) {
			return;
		}
		expandThreadForm();
		(document
			.querySelector("#new-thread-form input[type=file]") as any)
			.value = file;
		return
	}

	if (boardConfig.textOnly) {
		return
	}

	// Create form, if none
	postSM.feed(postEvent.open)

	// Either disconnected, erred or already has image
	const m = trigger("getPostModel") as FormModel
	if (m && !m.image) {
		m.uploadFile(file || files);
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
