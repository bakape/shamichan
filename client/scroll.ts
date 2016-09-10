// Various page scrolling aids

import {page} from "./state"
import options from "./options"

const $banner = document.querySelector("#banner") as HTMLElement
let $lock: HTMLElement,
	$reference: HTMLElement,
	atBottom: boolean

// Scroll to an element in the DOM with compensation for banner height
export function scrollToElement(el: HTMLElement) {
	const pos =
		el.getBoundingClientRect().top
		+ window.scrollY
		- $banner.offsetHeight
	window.scrollTo(0, pos)
}

// Scroll to target anchor element, if any
export function scrollToAnchor() {
	if (!location.hash) {
		return
	}
	scrollToElement(document.querySelector(location.hash) as HTMLElement)
}

// Lock position to the bottom of a thread or keep the viewport from bumping
// on out of sight DOM mutation.
export function followDOM(func: () => void) {
	const previous = referenceDistance()

	func()

	// Prevent scrolling with new posts, if page isn't visible
	if (atBottom && (!document.hidden || options.alwaysLock)) {
		window.scrollTo(0, document.body.scrollHeight)
	} else {
		// Only compensate, if the height increased above the viewport
		window.scrollBy(0, referenceDistance() - previous)
	}
}

// Set the scroll lock position to a post or to the bottom of the document
export function checkBottom() {
	if (!page.thread) {
		atBottom = false
		return
	}
	atBottom = window.scrollY + window.innerHeight >= document.body.scrollHeight
	if (!$lock) {
		$lock = document.querySelector("#lock") as HTMLElement
	}
	if ($lock) {
		$lock.style.visibility = atBottom ? "visible" : "hidden"
	}
}

// Returns distance of viewport to with current reference element
function referenceDistance(): number {
	return $reference.getBoundingClientRect().top
}

document.addEventListener("scroll", checkBottom, {
	passive: true,
})
