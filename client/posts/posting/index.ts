// Contains the FSM and core API for accessing the post authoring system

import FormModel from "./model"
import FormView from "./view"
import { connState, connSM } from "../../connection"
import { on, FSM, write, threads } from "../../util"
import lang from "../../lang"
import identity, { initIdentity } from "./identity"
import { boardConfig } from "../../state"
import initDrop from "./drop"
import initThreads from "./threads"

export { default as FormModel } from "./model"
export { default as identity } from "./identity"
export { spoilerImage } from "./upload"

// Sent to the FSM via the "open" and "hijack" events
export type FormMessage = {
	model: FormModel,
	view: FormView,
}

// Current post form view and model instances
export let postForm: FormView,
	postModel: FormModel

// Post authoring finite state machine
export const enum postState {
	none,    // No state. Awaiting first connection.
	ready,   // Ready to create posts
	halted,  // Post allocated to the server but no connectivity
	locked,  // No post open. Post creation controls locked.
	alloc,   // Post open and allocated to the server
	draft,   // Post open, but not yet allocated.
	errored, // Suffered unrecoverable error
}
export const enum postEvent {
	sync,       // Synchronized to the server
	disconnect, // Disconnected from server
	error,      // Unrecoverable error
	done,       // Post closed
	open,       // New post opened
	hijack,     // Hijacked an existing post as a postForm
	reset,      // Set to none. Used during page navigation.
	alloc,      // Allocated the draft post to the server
	reclaim,    // Ownership of post reclaimed after connectivity loss
	abandon,    // Abandon ownership of any open post
}
export const postSM = new FSM<postState, postEvent>(postState.none)

// Find the post creation button and style it, if any
function stylePostControls(fn: (el: HTMLElement) => void) {
	write(() => {
		const el = threads.querySelector("aside.posting")
		if (el) {
			fn(el)
		}
	})
}

// Ensures you are nagged at by the browser, when navigating away from an
// unfinished allocated post.
function bindNagging() {
	window.onbeforeunload = (event: BeforeUnloadEvent) =>
		event.returnValue = lang.ui["unfinishedPost"]
}

// Insert target post's number as a link into the text body. If text in the
// post is selected, quote it.
function quotePost(e: Event) {
	// Make sure the selection both starts and ends in the quoted post's
	// blockquote
	const post = (e.target as Element).closest("article"),
		gsel = getSelection()

	const isInside = (prop: string): boolean => {
		let el = gsel[prop] as HTMLElement
		if (!el) {
			return false
		}
		el = el.parentElement
		if (!el.closest("blockquote")) {
			return false
		}
		return el.closest("article") === post
	}

	let sel: string
	if (isInside("baseNode") && isInside("focusNode")) {
		sel = gsel.toString()
	}

	postSM.feed(postEvent.open)
	postModel.addReference(parseInt(post.id.slice(1)), sel)
}

// Update the draft post's fields on identity change, if any
function updateIdentity() {
	if (postSM.state === postState.draft && !boardConfig.forcedAnon) {
		postForm.renderIdentity()
	}
}

export default () => {
	// Synchronise with connection state machine
	connSM.on(connState.synced, postSM.feeder(postEvent.sync))
	connSM.on(connState.dropped, postSM.feeder(postEvent.disconnect))
	connSM.on(connState.desynced, postSM.feeder(postEvent.error))

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
	postSM.act(postState.halted, postEvent.reclaim, () => {
		postModel.flushBuffer()
		return postState.alloc
	})

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

	// Hijack and existing post and replace with post form and model
	postSM.act(
		postState.ready,
		postEvent.hijack,
		({view, model}: FormMessage) => {
			postModel = model
			postForm = view
			return postState.alloc
		},
	)

	postSM.on(postState.alloc, bindNagging)

	// Open a new post creation form, if none open
	postSM.act(postState.ready, postEvent.open, () => {
		postModel = new FormModel(0)
		postForm = new FormView(postModel, false)
		return postState.draft
	})

	// Hide post controls, when a postForm is open
	const hidePostControls = () =>
		stylePostControls(el =>
			el.style.display = "none")
	postSM.on(postState.draft, hidePostControls)
	postSM.on(postState.alloc, hidePostControls)

	// Close unallocated draft
	postSM.act(postState.draft, postEvent.done, () => {
		postForm.remove()
		return postState.ready
	})

	// Close allocated post
	postSM.act(postState.alloc, postEvent.done, () => {
		postModel.commitClose()
		return postState.ready
	})

	// Handle clicks on the [Reply] button
	on(threads, "click", postSM.feeder(postEvent.open), {
		selector: "aside.posting a",
	})

	// Handle clicks on post quoting links
	on(threads, "click", quotePost, {
		selector: "a.quote",
		passive: true,
	})

	// Trigger update on name or staff field change
	for (let id of ["name", "auth"]) {
		identity.onChange(id, updateIdentity)
	}

	initDrop()
	initThreads()
	initIdentity()
}
