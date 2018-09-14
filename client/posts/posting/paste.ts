// File upload or text alteration via clipboard paste
import { postSM, postEvent } from ".";
import { trigger, modPaste } from "../../util";
import { page, boardConfig } from "../../state";
import FormModel from "./model";
import { expandThreadForm } from "./threads";

// Handle file or text paste
function onPaste(e: ClipboardEvent) {
	const text = e.clipboardData.getData("text"),
	files = e.clipboardData.files
	var threadForm: HTMLFormElement,
	m: FormModel

	if (!text && files.length !== 1) {
		return
	}

	e.stopPropagation()
	e.preventDefault()

	if (!page.thread) {
		expandThreadForm()
		threadForm = document.querySelector("#new-thread-form") as HTMLFormElement
	} else {
		// Create form, if none
		postSM.feed(postEvent.open)
		// Neither disconnected, errored or already has image
		m = trigger("getPostModel") as FormModel
	}

	if (text) {
		if (threadForm) {
			const area = threadForm.querySelector("textarea[name=body]") as HTMLTextAreaElement,
			start = area.selectionStart,
			end = area.selectionEnd,
			old = area.value
			let p = modPaste(old, text, end)

			if (!p) {
				return
			}

			if (start != end) {
				area.value = old.slice(0, start) + p.body + old.slice(end)
				p.pos -= start
			} else {
				area.value = old.slice(0, end) + p.body + old.slice(end)
			}

			area.setSelectionRange(p.pos, p.pos)
			area.focus()
			return
		}

		if (m) {
			m.paste(text)
		}
	}

	if (files.length === 1) {
		if (boardConfig.textOnly) {
			return
		}

		if (threadForm) {
			(threadForm.querySelector("input[type=file]") as any).files = files
			return
		}

		if (m) {
			m.uploadFile(files.item(0))
		}
	}
}

// Bind listeners
export default () => document.addEventListener("paste", onPaste)
