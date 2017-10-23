// Various page scrolling aids

import { page } from "../state"
import { trigger } from "./hooks"

const banner = document.getElementById("banner")

let scrolled = false

// Indicates if the page is scrolled to its bottom
export let atBottom: boolean

// Scroll to target anchor element, if any
export function scrollToAnchor() {
	if (!location.hash) {
		if (!page.thread) {
			scrollToTop()
		}
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

// Scroll to the bottom of the thread
export function scrollToBottom() {
	window.scrollTo(0, document.documentElement.scrollHeight)
	atBottom = true
}

// Check, if at the bottom of the thread and render the locking indicator
export function checkBottom() {
	if (!page.thread) {
		atBottom = false
		return
	}
	atBottom = isAtBottom()
	const lock = document.getElementById("lock")
	if (lock) {
		lock.style.visibility = atBottom ? "visible" : "hidden"
	}
}

function isAtBottom(): boolean {
	return window.innerHeight + window.scrollY
		>= document.documentElement.offsetHeight
}

// Scrolled when the page is scrolled, unless it's at the bottom
document.addEventListener("scroll", () => {
	scrolled = !isAtBottom()
	atBottom = isAtBottom()
}, { passive: true })

// Use a MutationObserver to jump to the bottom of the page when a new
// post is made, unless the user has scrolled up from the bottom
let threadContainer = document.getElementById("thread-container")
if (threadContainer !== null) {
	let threadObserver = new MutationObserver((mut) => {
		if (!scrolled) {
			scrollToBottom()
		}
	})
	threadObserver.observe(threadContainer, {
		childList: true,
		subtree: true,
	})
}

// Unlock from bottom, when the tab is hidden, unless set not to
document.addEventListener("visibilitychange", () => {
	const opts = trigger("getOptions")
	if (document.hidden && (opts && !opts.alwaysLock)) {
		atBottom = false
	}
})

window.addEventListener("hashchange", scrollToAnchor, {
	passive: true,
})
