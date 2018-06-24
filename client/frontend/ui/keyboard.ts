// Keyboard shortcuts and such

import options from "../options"
import { FormModel, postSM, postEvent, toggleExpandAll, expandThreadForm } from "../posts"
import { page } from "../state"
import { trigger } from "../util"

// Bind keyboard event listener to the document
export default () =>
	document.addEventListener("keydown", handleShortcut)

function handleShortcut(event: KeyboardEvent) {
	let caught = false

	let anyModifier = event.altKey || event.metaKey || event.ctrlKey || event.shiftKey;
	let inInput = 'selectionStart' in event.target
	let altGr = event.getModifierState && event.getModifierState("AltGraph")

	if (!anyModifier && !inInput) {
		caught = true
		switch (event.key) {
			case "w":
			case "ArrowLeft":
				navigatePost(true)
				break
			case "s":
			case "ArrowRight":
				navigatePost(false)
				break
			default:
				caught = false
		}
	}

	if (event.altKey && !altGr) {
		caught = true

		switch (event.which) {
			case options.newPost:
				if (page.thread) {
					postSM.feed(postEvent.open)
				} else {
					expandThreadForm()
				}
				break
			case options.done:
				postSM.feed(postEvent.done, null)
				break
			case options.cancel:
				postSM.feed(postEvent.cancel, null)
				break
			case options.toggleSpoiler:
				const m = trigger("getPostModel") as FormModel
				if (m) {
					m.view.toggleSpoiler()
				}
				break
			case options.galleryMode:
				options.galleryModeToggle = !options.galleryModeToggle
				break
			case options.expandAll:
				toggleExpandAll()
				break
			case options.workMode:
				options.workModeToggle = !options.workModeToggle
				break
			case options.meguTVShortcut:
				options.meguTV = !options.meguTV
				break
			case 38:
				navigateUp()
				break
			default:
				caught = false
		}

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
		// Convert to absolute URL
		const a = document.createElement("a")
		a.href = url
		location.href = url
	}
}

const postSelector = "article[id^=p]"

// move focus to next or previous visible post in document order.
// starts with first post if none is selected via current url fragment
function navigatePost(reverse: boolean) {
	let all: Element[] = Array.from(document.querySelectorAll(postSelector))
	let current: Element = document.querySelector(postSelector + ":target") || all[0]
	let currentIdx = all.indexOf(current)

	while (current) {
		currentIdx = reverse ? currentIdx - 1 : currentIdx + 1
		current = all[currentIdx]
		if (current && window.getComputedStyle(current).display != "none") {
			break
		}
	}

	if (current) {
		window.location.hash = current.id
	}
}
