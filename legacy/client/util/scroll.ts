// Various page scrolling aids

import { page } from "../state"
import { trigger } from "./hooks"
import { lightenThread } from "../posts";

const banner = document.getElementById("banner")

let scrolled = false
let locked = false;

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
	const el = document.querySelector(location.hash) as HTMLElement
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
	const previous = atBottom;
	atBottom = isAtBottom()
	const lock = document.getElementById("lock")
	if (lock) {
		lock.style.visibility = atBottom ? "visible" : "hidden"
	}
	if (!previous && atBottom) {
		lightenThread();
	}
}

// Return, if scrolled to bottom of page
export function isAtBottom(): boolean {
	return window.innerHeight
		+ window.scrollY
		- document.documentElement.offsetHeight
		> -1
}

// If we are at the bottom, lock
document.addEventListener("scroll", () => {
	scrolled = !isAtBottom()
	locked = !scrolled;
	checkBottom();
}, { passive: true })

// Use a MutationObserver to jump to the bottom of the page when a new
// post is made, we are locked to the bottom or the user set the alwaysLock option
let threadContainer = document.getElementById("thread-container")
if (threadContainer !== null) {
	let threadObserver = new MutationObserver((mut) => {
		if (locked || (trigger("getOptions").alwaysLock && !scrolled)) {
			scrollToBottom()
		}
	})
	threadObserver.observe(threadContainer, {
		childList: true,
		subtree: true,
	})
}

// Unlock from bottom, when the tab is hidden
document.addEventListener("visibilitychange", () => {
	if (document.hidden) {
		locked = false
	}
})

window.addEventListener("hashchange", scrollToAnchor, {
	passive: true,
})
