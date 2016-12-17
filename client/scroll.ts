// Various page scrolling aids

import { page } from "./state"
import options from "./options"
import { deferInit } from "./defer"
import { threads } from "./render"

const banner = document.getElementById("banner")

let lock: HTMLElement,
	reference: Element,
	ticking: boolean

// Indicates if the page is scrolled to its bottom
export let atBottom: boolean

// Scroll to target anchor element, if any
export function scrollToAnchor() {
	if (!location.hash) {
		return
	}
	const el = document.querySelector(location.hash)
	if (!el) {
		return scrollToTop()
	}
	scrollToElement(el)
	checkBottom()
}

// Scroll to particular element and compensate for the banner height
export function scrollToElement(el: HTMLElement) {
	window.scrollTo(0, el.offsetTop - banner.offsetHeight - 5)
}

function scrollToTop() {
	window.scrollTo(0, 0)
	checkBottom()
}

// Lock position to the bottom of a thread or keep the viewport from bumping
// on out of sight DOM mutation.
export function followDOM(func: () => void) {
	// Don't compensate on board pages
	if (!page.thread) {
		return func()
	}

	const previous = referenceDistance()
	func()

	// Prevent scrolling with new posts, if page isn't visible
	if (atBottom) {
		return scrollToBottom()
	}
	// Element was removed or something
	if (!elExists(reference)) {
		return
	}

	// Only compensate, if the height increased above the viewport
	const delta = topDistance(reference, true) - previous
	if (delta) {
		window.scrollBy(0, delta)
	}
}

// Scroll to the bottom of the thread
export function scrollToBottom() {
	window.scrollTo(0, document.documentElement.scrollHeight)
	atBottom = true
}

// Set the scroll lock position to a post or to the bottom of the document
export function checkBottom() {
	if (!page.thread) {
		atBottom = false
		return
	}
	atBottom = window.innerHeight + window.scrollY
		>= document.documentElement.offsetHeight
	if (!lock) {
		lock = document.querySelector("#lock")
	}
	if (lock) {
		lock.style.visibility = atBottom ? "visible" : "hidden"
	}
}

// Check if element reference exists and is in the DOM
function elExists(el: Element): boolean {
	return !!el && document.contains(el)
}

// Return element position dimensions against the viewport, if the element
// is within the viewport
function topDistance(el: Element, skipCheck: boolean): number | null {
	const {top} = el.getBoundingClientRect()
	if (skipCheck || (top >= 0 && top < window.innerHeight)) {
		return top
	}
	return null
}

// Returns distance of viewport to current reference element
function referenceDistance(): number {
	if (elExists(reference)) {
		const bounds = topDistance(reference, false)
		if (bounds !== null) {
			return bounds
		}
	}

	// Find new reference element (first inside viewport). Account for empty
	// boards.
	for (let sel of ["article", "#threads"]) {
		for (let el of threads.querySelectorAll(sel)) {
			const bounds = topDistance(el, false)
			if (bounds !== null) {
				reference = el
				return bounds
			}
		}
	}
}

// Check, if we are at page bottom and persists to position on scroll. Deferred
// to animation frames to reduce lag.
function onScroll() {
	if (ticking) {
		return
	}
	ticking = true
	requestAnimationFrame(() => {
		checkBottom()
		ticking = false
	})
}

deferInit(() => {
	document.addEventListener("scroll", onScroll, {
		passive: true,
	})

	// Unlock from bottom, when the tab is hidden, unless set not to
	document.addEventListener("visibilitychange", () => {
		if (document.hidden && !options.alwaysLock) {
			atBottom = false
		}
	})
})
