import { message, send, handlers } from "../../connection"
import { Post } from "../model"
import { ImageData, PostData } from "../../common"
import FormView from "./view"
import { posts, storeMine, page, storeSeenPost } from "../../state"
import { postSM, postEvent, postState } from "."
import { extend } from "../../util"
import { SpliceResponse } from "../../client"
import { FileData } from "./upload"
import identity, { newAllocRequest } from "./identity"

// Form Model of an OP post
export default class FormModel extends Post {
	public sentAllocRequest: boolean
	public isAllocated: boolean

	// Disable live post updates
	public nonLive = (posts.get(page.thread) as any).nonLive || !identity.live

	public needCaptcha: boolean // Need to solve a captcha to allocate

	// Text that is not submitted yet to defer post allocation
	public bufferedText: string

	public inputBody = ""
	public view: FormView

	// Pass and ID, if you wish to hijack an existing model. To create a new
	// model pass zero.
	constructor() {
		// Initialize state
		super({
			id: 0,
			op: page.thread,
			editing: true,
			deleted: false,
			banned: false,
			sage: false,
			sticky: false,
			locked: false,
			time: Math.floor(Date.now() / 1000),
			body: "",
			name: "",
			auth: "",
			trip: "",
			state: {
				spoiler: false,
				quote: false,
				code: false,
				bold: false,
				italic: false,
				haveSyncwatch: false,
				successive_newlines: 0,
				iDice: 0,
			},
		})
	}

	// Append a character to the model's body and reparse the line, if it's a
	// newline
	public append(code: number) {
		if (this.editing) {
			this.body += String.fromCodePoint(code)
		}
	}

	// Remove the last character from the model's body
	public backspace() {
		if (this.editing) {
			this.body = this.body.slice(0, -1)
		}
	}

	// Splice the last line of the body
	public splice(msg: SpliceResponse) {
		if (this.editing) {
			this.spliceText(msg)
		}
	}

	// Compare new value to old and generate appropriate commands
	public parseInput(val: string): void {
		// Handle live update toggling
		if (this.nonLive) {
			this.bufferedText = val
			return
		}

		// Remove any buffered quote, as we are committing now
		this.bufferedText = ""

		const old = this.inputBody

		// Rendering hack shenanigans - ignore
		if (old === val) {
			return
		}

		const lenDiff = val.length - old.length,
			exceeding = old.length + lenDiff - 2000

		// If exceeding max body length, shorten the value, trim input and try
		// again
		if (exceeding > 0) {
			this.view.trimInput(exceeding)
			return this.parseInput(val.slice(0, -exceeding))
		}

		// Remove any lines past 30
		const lines = val.split("\n")
		if (lines.length - 1 > 100) {
			const trimmed = lines.slice(0, 100).join("\n")
			this.view.trimInput(val.length - trimmed.length)
			return this.parseInput(trimmed)
		}

		if (!this.sentAllocRequest) {
			this.requestAlloc(val, null)
		} else if (lenDiff === 1 && val.slice(0, -1) === old) {
			this.commitChar(val.slice(-1))
		} else if (lenDiff === -1 && old.slice(0, -1) === val) {
			this.commitBackspace()
		} else {
			this.commitSplice(val)
		}
	}

	// Commit a character appendage to the end of the line to the server
	private commitChar(char: string) {
		this.inputBody += char
		this.send(message.append, char.codePointAt(0))
	}

	// Optionally buffer all data, if currently disconnected
	private send(type: message, msg: any) {
		if (postSM.state !== postState.halted) {
			send(type, msg)
		}
	}

	// Send a message about removing the last character of the line to the
	// server
	private commitBackspace() {
		this.inputBody = this.inputBody.slice(0, -1)
		this.send(message.backspace, null)
	}

	// Commit any other input change that is not an append or backspace
	private commitSplice(v: string) {
		// Convert to arrays of chars to deal with multibyte unicode chars
		const old = [...this.inputBody],
			val = [...v],
			start = diffIndex(old, val),
			till = diffIndex(
				old.slice(start).reverse(),
				val.slice(start).reverse(),
			)

		this.send(message.splice, {
			start,
			len: old.length - till - start,
			// `|| undefined` ensures we never slice the string as [:0]
			text: val.slice(start, -till || undefined).join(""),
		})
		this.inputBody = v
	}

	// Close the form and revert to regular post
	public commitClose() {
		// It is possible to have never committed anything, if all you have in
		// the body is one quote and an image allocated.
		if (this.bufferedText) {
			this.nonLive = false
			this.parseInput(this.bufferedText)
		}

		this.body = this.inputBody
		this.abandon()
		this.send(message.closePost, null)
	}

	// Turn post form into a regular post, because it has expired after a
	// period of posting ability loss
	public abandon() {
		this.view.cleanUp()
		this.closePost()
	}

	// Add a link to the target post in the input
	public addReference(id: number, sel: string) {
		let s = ""
		const old = this.bufferedText || this.inputBody,
			newLine = !old || old.endsWith("\n")

		if (sel) {
			if (!newLine) {
				s += "\n"
			}
		} else if (!newLine && old[old.length - 1] !== " ") {
			s += " "
		}
		s += `>>${id} `

		if (!sel) {
			// If starting from a new line, insert newline after post link
			if (newLine) {
				s += "\n"
			}
		} else {
			s += "\n"
			for (let line of sel.split("\n")) {
				s += ">" + line + "\n"
			}
		}

		// Don't commit a quote, if it is the first input in a post
		let commit = true
		if (!this.sentAllocRequest && !this.bufferedText) {
			commit = false
		}
		this.view.replaceText(old + s, commit)

		// Makes sure the quote is committed later, if it is the first input in
		// the post
		if (!commit) {
			this.bufferedText = s
		}
	}

	// Commit a post made with live updates disabled
	public async commitNonLive() {
		let files: FileList
		if (this.view.upload) {
			files = this.view.upload.input.files
		}
		if (!this.bufferedText && !files.length) {
			return postSM.feed(postEvent.done)
		}

		this.sentAllocRequest = true

		const req = newAllocRequest()
		if (files.length) {
			req["image"] = await this.view.upload.uploadFile(files[0])
		}
		if (this.bufferedText) {
			req["body"] = this.body = this.bufferedText
		}

		send(message.insertPost, req)
		handlers[message.postID] = this.receiveID(false)
	}

	// Returns a function, that handles a message from the server, containing
	// the ID of the allocated post.
	// alloc specifies, if an alloc event should be fired on the state machine.
	private receiveID(alloc: boolean): (id: number) => void {
		return (id: number) => {
			this.id = id
			this.op = page.thread
			this.seenOnce = true
			if (alloc) {
				postSM.feed(postEvent.alloc)
			}
			storeSeenPost(this.id, this.op)
			storeMine(this.id, this.op)
			posts.add(this)
			delete handlers[message.postID]
		}
	}

	// Request allocation of a draft post to the server
	private requestAlloc(body: string | null, image: FileData | null) {
		const req = newAllocRequest()

		this.view.setEditing(true)
		this.nonLive = false
		this.sentAllocRequest = true

		req["open"] = !this.nonLive
		if (body) {
			req["body"] = body
			this.body = body
			this.inputBody = body
		}
		if (image) {
			req["image"] = image
		}

		send(message.insertPost, req)
		handlers[message.postID] = this.receiveID(true)
	}

	// Handle draft post allocation
	public onAllocation(data: PostData) {
		// May sometimes be called multiple times, because of reconnects
		if (this.isAllocated) {
			return
		}

		this.isAllocated = true
		extend(this, data)
		this.view.renderAlloc()
		if (data.image) {
			this.insertImage(this.image)
		}
		if (this.nonLive) {
			this.propagateLinks()
			postSM.feed(postEvent.done)
		}
	}

	// Upload the file and request its allocation
	public async uploadFile(files?: FileList) {
		if (files && this.view.upload) {
			(this.view.upload.input.files as any) = files
		}

		// Need a captcha and none submitted. Protects from no-captcha drops
		// allocating post too soon.
		if (this.needCaptcha || this.nonLive) {
			return
		}

		// Already have image or not in live mode
		if (this.image) {
			return
		}

		const data = await this.view.upload.uploadFile()
		// Upload failed, canceled, image added while thumbnailing or post
		// closed
		if (!data || this.image || !this.editing) {
			return
		}

		if (!this.sentAllocRequest) {
			this.requestAlloc(null, data)
		} else {
			send(message.insertImage, data)
		}
	}

	// Insert the uploaded image into the model
	public insertImage(img: ImageData) {
		this.image = img
		this.view.insertImage()
	}

	// Spoiler an already allocated image
	public commitSpoiler() {
		this.send(message.spoiler, null)
	}
}

// Find the first differing character in 2 character arrays
function diffIndex(a: string[], b: string[]): number {
	for (let i = 0; i < a.length; i++) {
		if (a[i] !== b[i]) {
			return i
		}
	}
	return a.length
}
