// Keyboard shortcuts and such

import options from "./options"
import { postForm, postSM, postEvent } from "./posts/posting/main"
import { toggleExpandAll } from "./posts/images"
import { page } from "./state"
import { scrollToElement } from "./scroll"
import navigate from "./history"

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
			if (page.thread) {
				postSM.feed(postEvent.open)
				break
			}
			const tf = document
				.querySelector("aside:not(.expanded) .new-thread-button")
			if (tf) {
				tf.click()
				scrollToElement(tf)
			}
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
		case options.workMode:
			options.workModeToggle = !options.workModeToggle
			break
		case 38:
			navigateUp()
			break
		default:
			caught = false
	}

	if (caught) {
		event.stopImmediatePropagation()
		event.preventDefault()
	}
}

// Navigate one level up the board tree, if possible
function navigateUp() {
	let url: string
	if (page.thread) {
		url = `/${page.board}/`
	} else if (page.board !== "all") {
		url = "/all/"
	}
	if (url) {
		navigate(url, null, true)
	}
}
