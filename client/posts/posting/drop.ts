// File upload via drag and drop

import { postSM, postEvent } from ".";
import { trigger } from "../../util";
import { page, boardConfig } from "../../state";
import FormModel from "./model";
import { expandThreadForm } from "./threads";

// Handle file drop
async function onDrop(e: DragEvent) {
	const { files } = e.dataTransfer;
	const url = e.dataTransfer.getData("text/uri-list");
	if ((!files.length && !url) || isFileInput(e.target)) {
		return;
	}

	e.stopPropagation();
	e.preventDefault();

	if (boardConfig.textOnly) {
		return;
	}

	if (!page.thread) {
		if (!files.length) {
			return;
		}
		expandThreadForm();
		(document
			.querySelector("#new-thread-form input[type=file]") as any)
			.files = files;
		return;
	}

	let file: File;
	if (files.length) {
		file = files[0];
	} else if (url) {
		// Fetch file from link
		try {
			let u = new URL(url);
			// Prevent URLs from meguca accidentally being posted with drag&drop
			if (u.origin === location.origin) {
				return;
			}
			const name = u.pathname.slice(u.pathname.lastIndexOf("/") + 1);
			file = new File([await (await fetch(url)).blob()], name);
		} catch (err) {
			alert(err);
			return;
		}
	}
	if (!file) {
		return;
	}

	// Create form, if none
	postSM.feed(postEvent.open);

	// Either disconnected, erred or already has image
	const m = trigger("getPostModel") as FormModel;
	if (m) {
		await m.uploadFile(file);
	}
}

// Returns, if event target is an <input type=file>
function isFileInput(target: EventTarget): boolean {
	const el = target as HTMLElement;
	return el.tagName === "INPUT" && el.getAttribute("type") === "file";
}

function stopDefault(e: Event) {
	if (!isFileInput(e.target)) {
		e.stopPropagation();
		e.preventDefault();
	}
}

// Bind listeners
export default () => {
	for (let event of ["dragenter", "dragexit", "dragover"]) {
		document.addEventListener(event, stopDefault);
	}
	document.addEventListener("drop", onDrop);
}
