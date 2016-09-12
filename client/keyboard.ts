// Keyboard shortcuts and such

import options from "./options"
import {postForm, postSM, postEvent} from "./posts/posting/main"
import {toggleExpandAll} from "./posts/images"

// Bind keyboard event listener to the document
export default function bindListener() {
	document.addEventListener("keydown", handleShortcut)
}

function handleShortcut(event: KeyboardEvent) {
	if (!event.altKey) {
		return
	}

	let caught = true
	switch (event.which) {
	case options.newPost:
		postSM.feed(postEvent.open)
		break
	case options.done:
		postSM.feed(postEvent.done)
		break
	case options.toggleSpoiler:
		if (postForm) {
			postForm.toggleSpoiler()
		}
		break
	case options.expandAll:
		toggleExpandAll()
		break

	// TODO: Work mode

	default:
		caught = false
	}


	if (caught) {
		event.stopImmediatePropagation()
		event.preventDefault()
	}
}
