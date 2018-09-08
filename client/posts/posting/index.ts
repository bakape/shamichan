// Contains the FSM and core API for accessing the post authoring system

import FormModel from "./model"
import FormView from "./view"
import { connState, connSM, handlers, message } from "../../connection"
import { on, FSM, hook } from "../../util"
import lang from "../../lang"
import identity, { initIdentity } from "./identity"
import { boardConfig, page } from "../../state"
import initDrop from "./drop"
import initPaste from "./paste"
import initFullScreen from "./fullscreen"
import initImageErr from "./image"
import initThreads from "./threads"
import { renderCaptchaForm } from "./captcha";

export { default as FormModel } from "./model"
export { default as identity } from "./identity"
export { expandThreadForm } from "./threads"

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
	lastSelection: Selection

// Post authoring finite state machine
export const enum postState {
	// No state. Awaiting first connection.
	none,
	// Ready to create posts
	ready,
	// Post allocated to the server but no connectivity
	halted,
	// No post open. Post creation controls locked.
	locked,
	// Post open and allocated to the server
	alloc,
	// Post open, but not yet allocating
	draft,
	// Sent a request to allocate a live post
	allocating,
	// Suffered unrecoverable error
	erred,
	// Post creation disabled in thread
	threadLocked,
}
export const enum postEvent {
	// Synchronized to the server
	sync,
	// Disconnected from server
	disconnect,
	// Unrecoverable error
	error,
	// Post closed
	done,
	// New post opened
	open,
	// Set to none. Used during page navigation.
	reset,
	// A live post allocation request has been sent to the server
	sentAllocRequest,
	// Allocated the draft post to the server
	alloc,
	// Ownership of post reclaimed after connectivity loss
	reclaim,
	// Abandon ownership of any open post
	abandon,
	// Server requested to solve a captcha
	captchaRequested,
	// Captcha successfully solved
	captchaSolved,
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
	handlers[message.captcha] = postSM.feeder(postEvent.captchaRequested);

	// Initial synchronization
	postSM.act(postState.none, postEvent.sync, () =>
		postState.ready)

	// Set up client to create new posts
	postSM.on(postState.ready, () => {
		if (postModel) {
			postModel.abandon();
		}

		// Don't null postForm. It may still be modified mid-transition
		window.onbeforeunload = postModel = null

		stylePostControls(el => {
			el.style.display = ""
			el.classList.remove("disabled")
		})
	})

	// Update Done button on any state change
	postSM.onChange(() => {
		if (postForm) {
			postForm.updateDoneButton();
		}
	});

	// Handle connection loss
	postSM.wildAct(postEvent.disconnect, () => {
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
	postSM.act(postState.halted, postEvent.abandon, () =>
		postState.ready);

	// Regained connectivity, when no post open
	postSM.act(postState.locked, postEvent.sync, () =>
		postState.ready)

	// Handle critical errors
	postSM.wildAct(postEvent.error, () => {
		stylePostControls(el =>
			el.classList.add("erred"))
		postForm && postForm.renderError()
		window.onbeforeunload = null
		return postState.erred
	})

	// Reset state during page navigation
	postSM.wildAct(postEvent.reset, () =>
		postState.ready)

	// Transition a draft post into allocated state. All the logic for this is
	// model- and view-side.
	postSM.act(postState.allocating, postEvent.alloc, () =>
		postState.alloc);

	postSM.on(postState.alloc, bindNagging)

	// Open a new post creation form, if none open
	postSM.act(postState.ready, postEvent.open, () => {
		postModel = new FormModel()
		postForm = new FormView(postModel)
		return postState.draft
	})

	// Hide post controls, when a postForm is open
	const hidePostControls = () =>
		stylePostControls(el =>
			el.style.display = "none")
	postSM.on(postState.draft, hidePostControls)
	postSM.on(postState.alloc, () =>
		hidePostControls())

	postSM.act(postState.draft, postEvent.sentAllocRequest, () =>
		postState.allocating);

	// Close unallocated draft or commit in non-live mode
	postSM.act(postState.draft, postEvent.done, () => {
		postForm.remove();
		return postState.ready;
	})

	// Server requested captcha. This rejects the previous post or image
	// allocation request.
	for (let s of [postState.draft, postState.allocating]) {
		postSM.act(s, postEvent.captchaRequested, () => {
			postModel.inputBody = "";
			renderCaptchaForm();
			if (postForm.upload) {
				postForm.upload.reset();
			}
			return postState.draft;
		});
	}
	postSM.act(postState.alloc, postEvent.captchaRequested, () => {
		renderCaptchaForm();
		if (postForm.upload) {
			postForm.upload.reset();
		}
		return postState.alloc;
	})

	// Attempt to resume post after solving captcha
	for (let s of [postState.draft, postState.alloc]) {
		const _s = s; // Persist variable in inner scope
		postSM.act(_s, postEvent.captchaSolved, () => {
			postModel.retryUpload();
			postForm.input.focus();
			return _s;
		});
	}

	// Close allocated post
	postSM.act(postState.alloc, postEvent.done, () => {
		postModel.commitClose();
		return postState.ready;
	});

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

	// Toggle live update committing on the input form, if any
	identity.onChange("live", (live: boolean) => {
		if (postSM.state !== postState.draft) {
			return;
		}
		postForm.setEditing(live);
		postForm.inputElement("done").hidden = live;
	});

	initDrop()
	initPaste()
	initFullScreen()
	initImageErr()
	initThreads()
	initIdentity()
}
