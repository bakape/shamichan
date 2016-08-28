// Utility functions for reducing layout thrashing, by batching DOM writes and
// reads. Basically a stripped down version of FastDOM.
// Also contains utilities for HTML template tags.

import * as lang from './lang'

type Operation = () => void

// Cached element containing the banners, posts, catalog and other board HTML
export const $threads = document.querySelector("#threads")

// Holds cached references to all out HTML template tags' contents
const templates: {[name: string]: DocumentFragment} = {}

let readStack: Operation[] = [],
	writeStack: Operation[] = [],
	scheduled: boolean

// Initialize and populate templates with language pack values
for (let el of document.head.querySelectorAll("template")) {
	templates[el.getAttribute("name")] = (el as HTMLTemplateElement).content
}
{
	const frag = templates["board"]
	for (let el of frag.querySelectorAll(".new-thread-button")) {
		el.textContent = lang.posts.newThread
	}
}
{
	const frag = templates["thread"]
	const actText = [
		lang.navigation.bottom,
		lang.images.expand,
		lang.navigation.return,
		lang.navigation.top,
	]
	const actEls = frag.querySelectorAll("span a")
	for (let i = 0; i < actEls.length; i++) {
		actEls[i].textContent = actText[i]
	}
	frag.querySelector("aside.posting a").textContent = lang.posts.reply
	frag.querySelector("#lock").textContent = lang.navigation.lockedToBottom
}
{
	const frag = templates["catalog-thread"],
		links = frag.querySelector(".thread-links")
	links
		.querySelector(".counters")
		.setAttribute("title", lang.navigation.catalogOmit)
}
{
	const frag = templates["form"]
	; (frag.querySelector("input[type=submit]") as HTMLInputElement)
		.value = lang.ui.submit
	; (frag.querySelector("input[name=cancel]") as HTMLInputElement)
		.value = lang.ui.cancel
	frag.querySelector(".captcha-image")
		.setAttribute("title", lang.ui.reloadCaptcha)
	frag.querySelector("input[name=adcopy_response]")
		.setAttribute("placeholder", lang.ui.focusForCaptcha)
}

// Import a prepared template and return it's HTML contents
export function importTemplate(name: string): DocumentFragment {
	return document.importNode(templates[name], true) as DocumentFragment
}

// Schedule a DOM write operation
export function write(operation: Operation) {
	writeStack.push(operation)
	scheduleFlush()
}

// Schedule a DOM read operation
export function read(operation: Operation) {
	readStack.push(operation)
	scheduleFlush()
}

// Schedule a flush on the next animation frame, if not yet scheduled
function scheduleFlush() {
	if (!scheduled) {
		scheduled = true
		requestAnimationFrame(flush)
	}
}

// Perform all write tasks and then read tasks in the stack
function flush() {
	const writes = writeStack,
		reads = readStack
	writeStack = []
	readStack = []
	scheduled = false
	for (let i = 0; i < writes.length; i++) {
		writes[i]()
	}
	for (let i = 0; i < reads.length; i++) {
		reads[i]()
	}
}
