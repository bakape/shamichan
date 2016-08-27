// Logic for manipulating the views and FSM of post authoring and communicated
// the results to the server

import {message, send} from "../../connection"
import {OP, Post, TextState, ThreadData} from "../models"
import {FormView, OPFormView} from "./view"
import {posts} from "../../state"
import {postSM, postEvent, postState} from "./main"
import {applyMixins} from "../../util"
import PostView from "../view"
import {SpliceResponse} from "../../client"

type BufferedMessage = [message, any]

// Form Model of an OP post
export class OPFormModel extends OP implements FormModel {
	bodyLength: number
	parsedLines: number
	view: FormView
	inputState: TextState
	messageBuffer: BufferedMessage[]

	commitChar: (char: string) => void
	commitBackspace: () => void
	commitClose: () => void
	commitSplice: (val: string) => void
	flushBuffer: () => void
	init: () => void
	lastBodyLine: () => string
	parseInput: (val: string) => void
	send: (type: message, msg: any) => void

	constructor(id: number) {

		// TODO: Persist id to state.mine

		const oldModel = posts.get(id) as OP,
			oldView = oldModel.view
		oldView.unbind()

		// Copy the parent model's state and data
		super(extractAttrs(oldModel) as ThreadData)

		// Replace old model and view pair with the postForm pair
		posts.addOP(this)
		const view = new OPFormView(this)
		oldView.el.replaceWith(view.el)
		postSM.feed(postEvent.hijack, {view, model: this})

		this.init()
	}
}

// Form model for regular reply posts
export class ReplyFormModel extends Post implements FormModel {
	bodyLength: number
	parsedLines: number
	view: FormView
	inputState: TextState
	messageBuffer: BufferedMessage[]

	commitChar: (char: string) => void
	commitBackspace: () => void
	commitClose: () => void
	commitSplice: (val: string) => void
	flushBuffer: () => void
	init: () => void
	lastBodyLine: () => string
	parseInput: (val: string) => void
	send: (type: message, msg: any) => void

	constructor() {
		super({
			id: 0,
			editing: true,
			time: Date.now(),
			body: "",
			state: {
				spoiler: false,
				quote: false,
				iDice: 0,
			},
		})

		this.init()
	}
}

// Override mixin for post authoring models
export class FormModel {
	bodyLength: number = 0 // Compound length of the input text body
	parsedLines: number = 0 // Number of closed, commited and parsed lines
	body: string
	view: PostView & FormView
	state: TextState // State of the underlying normal post model

	// State of line being edditted. Must be seperated to not affect the
	// asynchronous updates of commited lines
	inputState: TextState

	// Buffer for messages commited during connection outage
	messageBuffer: BufferedMessage[]

	spliceLine: (line: string, msg: SpliceResponse) => string
	resetState: () => void

	// Initialize state
	init() {
		this.bodyLength = this.parsedLines = 0
		this.inputState = {
			quote: false,
			spoiler: false,
			iDice: 0, // Not used in FormModel. TypeScipt demands it.
			line: "",
		}
	}

	// Append a character to the model's body and reparse the line, if it's a
	// newline
	append(code: number) {
		const char = String.fromCharCode(code)
		if (char === "\n") {
			this.view.terminateLine(this.parsedLines++)
		}
		this.body += char
	}

	// Remove the last character from the model's body
	backspace() {
		this.body = this.body.slice(0, -1)
	}

	// Splice the last line of the body
	splice(msg: SpliceResponse) {
		this.spliceLine(this.lastBodyLine(), msg)
	}

	// Compare new value to old and generate apropriate commands
	parseInput(val: string): void {
		const old = this.inputState.line,
			lenDiff = val.length - old.length,
			exceeding = this.bodyLength + lenDiff - 2000

		// If exceeding max body lenght, shorten the value, trim $input and try
		// again
		if (exceeding > 0) {
			this.view.trimInput(exceeding)
			return this.parseInput(val.slice(0, -exceeding))
		}

		if (lenDiff === 1 && val.slice(0, -1) === old) {
			return this.commitChar(val.slice(-1))
		}
		if (lenDiff === -1 && old.slice(0, -1) === val) {
			return this.commitBackspace()
		}

		return this.commitSplice(val, lenDiff)
	}

	// Commit a character appendage to the end of the line to the server
	commitChar(char: string) {
		this.bodyLength++
		if (char === "\n") {
			this.resetState()
			this.view.startNewLine()
			this.inputState.line = ""
		} else {
			this.inputState.line += char
		}
		this.send(message.append, char.charCodeAt(0))
	}

	// Optionally buffer all data, if currently disconnected
	send(type: message, msg: any) {
		if (postSM.state === postState.halted) {
			this.messageBuffer.push([type, msg])
		} else {
			send(type, msg)
		}
	}

	// Flush any buffered messages to the server
	flushBuffer() {
		for (let [type, msg] of this.messageBuffer) {
			send(type, msg)
		}
		this.messageBuffer = []
	}

	// Send a message about removing the last character of the line to the
	// server
	commitBackspace() {
		this.inputState.line = this.inputState.line.slice(0, -1)
		this.bodyLength--
		this.send(message.backspace, null)
	}

	// Commit any other $input change that is not an append or backspace
	commitSplice(val: string, lenDiff: number) {
		const old = this.inputState.line
		let start: number,
			len: number,
			text: string

		// Find first differing character
		for (let i = 0; i < old.length; i++) {
			if (old[i] !== val[i]) {
				start = i
				break
			}
		}

		// Find last common character and the differing part
		const maxLen = Math.max(old.length, val.length),
			vOffset = val.length - maxLen,
			oOffset = old.length - maxLen
		for (let i = maxLen; i >= start; i--) {
			if (old[i + oOffset] !== val[i + vOffset]) {
				len = i + oOffset - start + 1
				text = val.slice(start).slice(0, len - 1)
				break
			}
		}

		this.send(message.splice, {start, len, text})
		this.bodyLength += lenDiff
		this.inputState.line = val

		// If splice contained newlines, reformat text accordingly
		const lines = val.split("\n")
		if (lines.length > 1) {
			const lastLine = lines[lines.length - 1]
			this.view.injectLines(lines.slice(0, -1), lastLine)
			this.resetState()
			this.inputState.line = lastLine
		}
	}

	// Close the form and revert to regular post
	commitClose() {

		// TODO: Need some warning, if closing a post, when there is no
		// connectivity. This might become very confusing otherwisse.

		// Normalize state
		this.state.line = this.inputState.line
		postSM.feed(postEvent.done)
		this.view.cleanUp()
		this.send(message.closePost, null)
		postSM.feed(postEvent.done)
	}

	// Return the last line of the body
	lastBodyLine(): string {
		const lines = this.body.split("\n")
		return lines[lines.length - 1]
	}
}

applyMixins(OPFormModel, FormModel)
applyMixins(ReplyFormModel, FormModel)

// Extract all non-function attributes from a model
function extractAttrs(src: {[key: string]: any}): {[key: string]: any} {
	const attrs: {[key: string]: any} = {}
	for (let key in src) {
		if (typeof src[key] !== "function") {
			attrs[key] = src[key]
		}
	}
	return attrs
}
