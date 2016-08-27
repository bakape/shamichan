// Contains the FSM and core API for accessing the post authoring system

import {FormModel, ReplyFormModel} from "./model"
import {Post} from "../models"
import {FormView} from "./view"
import FSM from "../../fsm"
import {connState, connSM} from "../../connection"
import {write, $threads} from "../../render"
import {posts as lang} from "../../lang"
import {on} from "../../util"

// Sent to the FSM via the "open" and "hijack" events
export type FormMessage = {
	model: FormModel & Post,
	view: FormView,
}

// Current post form view and model instances
export let postForm: FormView
export let postModel: FormModel & Post

// Post authoring finite state machine
export const enum postState {
	none,    // No state. Awating first connection.
	ready,   // Ready to create posts
	halted,  // Post allocated to thhe server but no connection
	locked,  // No post open. Post creation controls locked.
	alloc,   // Post open and allocated to the server
	draft,   // Post open, but not yet allocated.
	errored, // Suffered unrecoverable error
}
export const enum postEvent {
	sync,       // Synchronised to the server
	disconnect, // Disconnected from server
	error,      // Unrecoverable error
	done,       // Post closed
	open,       // New post opened
	hijack,     // Hijacked an existing post as a postForm
	reset,      // Set to none. Used during page navigation.
	alloc,      // Allocated the draft post to the server
}
export const postSM = new FSM<postState, postEvent>(postState.none)

// Synchronise with connection state machine
connSM.on(connState.synced, postSM.feeder(postEvent.sync))
connSM.on(connState.dropped, postSM.feeder(postEvent.disconnect))
connSM.on(connState.desynced, postSM.feeder(postEvent.error))

// Find the post creation button and style it
const stylePostControls = (fn: (el: HTMLElement) => void) =>
	write(() =>
		fn($threads.querySelector("aside.posting") as HTMLElement))

// Handle connection loss
postSM.wildAct(postEvent.disconnect, () => {
	switch (postSM.state) {
	case postState.alloc:       // Pause current allocated post
		return postState.halted
	case postState.draft:       // Clear any unallocated postForm
		postForm.remove()
		postModel = postForm = null
		stylePostControls(el =>
			el.style.display = "")
	}

	stylePostControls(el =>
		el.classList.add("disabled"))

	return postState.locked
})

// Regained conectitvity, when post is open
postSM.act(postState.halted, postEvent.sync, () =>
	(postModel.flushBuffer(),
	postState.alloc))

// Regained connectivity, when no post open
postSM.act(postState.locked, postEvent.sync, () =>
	postState.ready)

// Handle critical errors
postSM.wildAct(postEvent.error, () =>
	(stylePostControls(el =>
		el.classList.add("errored")),
	postForm && postForm.renderError(),
	postState.errored))

// Reset state during page navigation
postSM.wildAct(postEvent.reset, () =>
	(resetState(),
	postState.ready))

// Reset module state to initial
function resetState() {
	window.onbeforeunload = postForm = postModel = null
}

// Transition a draft post into allocated state. All the logic for this is
// model- and view-side.
postSM.act(postState.draft, postEvent.alloc, () =>
	postState.alloc)

// Hijack and existing post and replace with post form and model
postSM.act(postState.ready, postEvent.hijack, ({view, model}: FormMessage) =>
	(postModel = model,
	postForm = view,
	postState.alloc))

postSM.on(postState.alloc, bindNagging)

// Ensures you are nagged at by the browser, when navigating away from an
// unfinished allocated post.
function bindNagging() {
	window.onbeforeunload = (event: BeforeUnloadEvent) =>
		event.returnValue = lang.unfinishedPost
}

// Open a new post creation form, if none open
postSM.act(postState.ready, postEvent.open, () =>
	(postModel = new ReplyFormModel(),
	postForm = new FormView(postModel, false),
	postState.alloc))

// Hide post controls, when a postForm is open
const hidePostControls = () =>
	stylePostControls(el =>
		el.style.display = "none")
postSM.on(postState.draft, hidePostControls)
postSM.on(postState.alloc, hidePostControls)

// Register all transitions that lead to postState.ready
const toReady = () =>
	(resetState(),
	postState.ready)
const readyTransitions: [postState, postEvent][] = [
	[postState.none, postEvent.sync],
	[postState.draft, postEvent.done],
	[postState.alloc, postEvent.done],
]
for (let [state, event] of readyTransitions) {
	postSM.act(state, event, toReady)
}
postSM.on(postState.ready, () =>
	stylePostControls(el =>
		(el.style.display = "",
		el.classList.remove("disabled"))))

// Handle clicks on the [Reply] button
on($threads, "click", postSM.feeder(postEvent.open), {
	selector: "aside.posting a",
})
