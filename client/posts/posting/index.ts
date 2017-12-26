// Contains the FSM and core API for accessing the post authoring system

import FormModel from "./model"
import FormView from "./view"
import { connState, connSM, handlers, message } from "../../connection"
import { on, FSM, hook } from "../../util"
import lang from "../../lang"
import identity, { initIdentity } from "./identity"
import { boardConfig, page } from "../../state"
import initDrop from "./drop"
import initThreads from "./threads"
import options from "../../options"

export { default as FormModel } from "./model"
export { default as identity } from "./identity"
export { expandThreadForm } from "./threads"

// Sent to the FSM via the "open" and "hijack" events
export type FormMessage = {
	model: FormModel,
	view: FormView,
}

type Selection = {
	start: Node
	end: Node
	text: string
}

// Current post form view and model instances
let postForm: FormView,
	postModel: FormModel,
	// Store last selected range, so we can access it after a mouse click on
	// quote links, which cause that link to become selected
	lastSelection: Selection,
	// Specifies, if a captcha solved is needed to allocate a post
	needCaptcha = false

// Post authoring finite state machine
export const enum postState {
	none,           // No state. Awaiting first connection.
	ready,          // Ready to create posts
	halted,         // Post allocated to the server but no connectivity
	locked,         // No post open. Post creation controls locked.
	alloc,          // Post open and allocated to the server
	draft,          // Post open, but not yet allocated.
	needCaptcha,    // Awaiting a captcha to be solved
	sendingNonLive, // Sending a request to a allocate a post in non-live mode
	errored,        // Suffered unrecoverable error
	threadLocked,   // Post creation disabled in thread
}
export const enum postEvent {
	sync,          // Synchronized to the server
	disconnect,    // Disconnected from server
	error,         // Unrecoverable error
	done,          // Post closed
	open,          // New post opened
	reset,         // Set to none. Used during page navigation.
	alloc,         // Allocated the draft post to the server
	reclaim,       // Ownership of post reclaimed after connectivity loss
	abandon,       // Abandon ownership of any open post
	captchaSolved, // New captcha solved and submitted
}
export const postSM = new FSM<postState, postEvent>(postState.none)

hook("getPostModel", () =>
	postModel)

// Find the post creation button and style it, if any
function stylePostControls(fn: (el: HTMLElement) => void) {
	const el = document.querySelector("aside.posting")
	if (el) {
		fn(el)
	}
}

// Ensures you are nagged at by the browser, when navigating away from an
// unfinished allocated post.
function bindNagging() {
	window.onbeforeunload = (event: BeforeUnloadEvent) =>
		event.returnValue = lang.ui["unfinishedPost"]
}

// Insert target post's number as a link into the text body. If text in the
// post is selected, quote it.
function quotePost(e: MouseEvent) {
	// Don't trigger, when user is trying to open in a new tab
	const bypass = e.which !== 1
		|| e.ctrlKey
		|| (page.thread && connSM.state !== connState.synced)
	if (bypass) {
		return
	}

	const target = e.target as HTMLAnchorElement

	// Make sure the selection both starts and ends in the quoted post's
	// blockquote
	const post = target.closest("article")
	const isInside = (prop: string): boolean => {
		const node = lastSelection[prop] as Node
		if (!node) {
			return false
		}
		const el = node.nodeType === Node.TEXT_NODE
			? node.parentElement
			: node as Element
		if (!el) { // No idea why, but el sometimes is null
			return false
		}

		// Selection bound is mid-post
		if (el.closest("blockquote") && el.closest("article") === post) {
			return true
		}
		switch (prop) {
			// Selection start at blockquote start
			case "start":
				return el === post
			// Selection end is at blockquote end
			case "end":
				if (el.closest("article") === post.nextSibling) {
					return true
				}
				if (el.tagName === "SECTION") {
					// Avoids capturing the [Reply] button
					const i = lastSelection.text.lastIndexOf("\n")
					if (i >= 0) {
						lastSelection.text = lastSelection.text.slice(0, i)
					}
					return true
				}
				return false
		}
	}
	let sel = ""
	if (lastSelection && isInside("start") && isInside("end")) {
		sel = lastSelection.text
	}

	const id = parseInt(post.id.slice(1))

	// On board pages, first navigate to the thread
	if (!page.thread) {
		location.href = target.href

		// Store, so a reply is opened, when the page is loaded
		localStorage.setItem("openQuote", `${id}:${sel}`)
		return
	}

	postSM.feed(postEvent.open)
	postModel.addReference(id, sel)
}

// Update the draft post's fields on identity change, if any
function updateIdentity() {
	if (postSM.state === postState.draft && !boardConfig.forcedAnon) {
		postForm.renderIdentity()
	}
}

// Toggle live update committing on the input form, if any
function toggleLive(live: boolean) {
	if (!postModel || postModel.sentAllocRequest) {
		return
	}
	postForm.setEditing(live)
	postForm.inputElement("done").hidden = live
	postModel.nonLive = !live
}

async function openReply(e: MouseEvent) {
	// Don't trigger, when user is trying to open in a new tab
	if (e.which !== 1
		|| !page.thread
		|| e.ctrlKey
		|| connSM.state !== connState.synced
	) {
		return
	}

	e.preventDefault()
	postSM.feed(postEvent.open)
}

export default () => {
	// Synchronise with connection state machine
	connSM.on(connState.synced, postSM.feeder(postEvent.sync))
	connSM.on(connState.dropped, postSM.feeder(postEvent.disconnect))
	connSM.on(connState.desynced, postSM.feeder(postEvent.error))

	// The server notified a captcha will be required on the next post
	handlers[message.captcha] = () =>
		needCaptcha = true

	// Initial synchronization
	postSM.act(postState.none, postEvent.sync, () =>
		postState.ready)

	// Set up client to create new posts
	postSM.on(postState.ready, () => {
		window.onbeforeunload = postForm = postModel = null
		stylePostControls(el => {
			el.style.display = ""
			el.classList.remove("disabled")
		})
	})

	// Handle connection loss
	postSM.wildAct(postEvent.disconnect, () => {
		needCaptcha = false

		switch (postSM.state) {
			case postState.alloc:       // Pause current allocated post
			case postState.halted:
				return postState.halted
			case postState.draft:       // Clear any unallocated postForm
				postForm.remove()
				postModel = postForm = null
				stylePostControls(el =>
					el.style.display = "")
				break
			case postState.locked:
				return postState.locked
		}

		stylePostControls(el =>
			el.classList.add("disabled"))

		return postState.locked
	})

	// Regained connectivity, when post is allocated
	postSM.act(postState.halted, postEvent.reclaim, () =>
		postState.alloc)

	// Regained connectivity too late and post can no longer be reclaimed
	postSM.act(postState.halted, postEvent.abandon, () => {
		postModel.abandon()
		return postState.ready
	})

	// Regained connectivity, when no post open
	postSM.act(postState.locked, postEvent.sync, () =>
		postState.ready)

	// Handle critical errors
	postSM.wildAct(postEvent.error, () => {
		stylePostControls(el =>
			el.classList.add("errored"))
		postForm && postForm.renderError()
		window.onbeforeunload = null
		return postState.errored
	})

	// Reset state during page navigation
	postSM.wildAct(postEvent.reset, () =>
		postState.ready)

	// Transition a draft post into allocated state. All the logic for this is
	// model- and view-side.
	postSM.act(postState.draft, postEvent.alloc, () =>
		postState.alloc)

	postSM.on(postState.alloc, bindNagging)

	// Open a new post creation form, if none open
	postSM.act(postState.ready, postEvent.open, () => {
		postModel = new FormModel()
		postModel.needCaptcha = needCaptcha
		postForm = new FormView(postModel)
		if (needCaptcha) {
			return postState.needCaptcha
		}
		return postState.draft
	})

	// New captcha submitted
	postSM.act(postState.needCaptcha, postEvent.captchaSolved, () => {
		postModel.needCaptcha = needCaptcha = false
		if ((postForm.upload && postForm.upload.input.files.length)
			&& !postModel.nonLive
		) {
			postModel.uploadFile()
		}
		return postState.draft
	})

	// Cancelled, when needing a captcha
	postSM.act(postState.needCaptcha, postEvent.done, () => {
		postForm.remove()
		return postState.ready
	})

	// Hide post controls, when a postForm is open
	const hidePostControls = () =>
		stylePostControls(el =>
			el.style.display = "none")
	postSM.on(postState.draft, hidePostControls)
	postSM.on(postState.alloc, () =>
		hidePostControls())

	// Close unallocated draft
	postSM.act(postState.draft, postEvent.done, (e?: Event) => {
		// Commit a draft made as a non-live post
		let commitNonLive = false
		if (e && postModel.nonLive) {
			if (e.target instanceof HTMLInputElement) {
				commitNonLive = e.target.getAttribute("name") === "done"
			} else if (e instanceof KeyboardEvent) {
				commitNonLive = e.which === options.done
			}
		}
		if (commitNonLive) {
			if (postModel.needCaptcha) { // New captcha submitted
				needCaptcha = false
			}
			postModel.commitNonLive()
			return postState.sendingNonLive
		}

		postForm.remove()
		return postState.ready
	})

	// Close allocated post
	postSM.act(postState.alloc, postEvent.done, () => {
		postModel.commitClose()
		return postState.ready
	})

	// Just close the post, after it is committed
	postSM.act(postState.sendingNonLive, postEvent.done, () => {
		postModel.abandon()
		return postState.ready
	})

	// Handle clicks on the [Reply] button
	on(document, "click", openReply, {
		selector: "aside.posting a",
	})

	// Handle clicks on post quoting links
	on(document, "click", quotePost, {
		selector: "a.quote",
	})

	// Store last selected range that is not a quote link
	document.addEventListener("selectionchange", () => {
		const sel = getSelection(),
			start = sel.anchorNode
		if (!start) {
			return
		}
		const el = start.parentElement
		if (el && !el.classList.contains("quote")) {
			lastSelection = {
				start: sel.anchorNode,
				end: sel.focusNode,
				text: sel.toString().trim(),
			}
		}
	})

	// Trigger post form updates on post option change
	for (let id of ["name", "auth", "sage"]) {
		identity.onChange(id, updateIdentity)
	}
	identity.onChange("live", toggleLive)

	initDrop()
	initThreads()
	initIdentity()
}
