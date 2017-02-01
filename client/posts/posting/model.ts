import { message, send, handlers } from "../../connection"
import { Post } from "../model"
import { ImageData, PostData } from "../../common"
import FormView from "./view"
import { posts, storeMine } from "../../state"
import { postSM, postEvent, postState } from "."
import { extend, write } from "../../util"
import { SpliceResponse } from "../../client"
import { FileData } from "./upload"
import { newAllocRequest } from "./identity"

// Form Model of an OP post
export default class FormModel extends Post {
	private sentAllocRequest: boolean
	public isAllocated: boolean
	private inputBody: string
	public view: FormView
	private lasLinked: number // ID of last linked post

	// Pass and ID, if you wish to hijack an existing model. To create a new
	// model pass zero.
	constructor(id: number) {
		if (id !== 0) {
			storeMine(id)

			const oldModel = posts.get(id),
				oldView = oldModel.view
			oldView.unbind()

			// Copy the parent model's state and data
			const attrs = {} as PostData
			for (let key in oldModel) {
				if (typeof oldModel[key] !== "function") {
					attrs[key] = oldModel[key]
				}
			}
			super(attrs)

			// Replace old model and view pair with the postForm pair
			posts.add(this)
			const view = new FormView(this, true)
			oldView.el.replaceWith(view.el)

			postSM.feed(postEvent.hijack, { view, model: this })
			this.sentAllocRequest = this.isAllocated = true
		} else {
			super({
				id: 0,
				editing: true,
				time: Date.now(),
				body: "",
				state: {
					spoiler: false,
					quote: false,
					lastLineEmpty: false,
					iDice: 0,
				},
			})
		}

		// Initialize state
		this.inputBody = ""
	}

	// Append a character to the model's body and reparse the line, if it's a
	// newline
	public append(code: number) {
		this.body += String.fromCodePoint(code)
	}

	// Remove the last character from the model's body
	public backspace() {
		this.body = this.body.slice(0, -1)
	}

	// Splice the last line of the body
	public splice(msg: SpliceResponse) {
		this.body = this.spliceText(this.body, msg)
	}

	// Compare new value to old and generate appropriate commands
	public parseInput(val: string): void {
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

		if (!this.sentAllocRequest) {
			return this.requestAlloc(val, null)
		}

		if (lenDiff === 1 && val.slice(0, -1) === old) {
			return this.commitChar(val.slice(-1))
		}
		if (lenDiff === -1 && old.slice(0, -1) === val) {
			return this.commitBackspace()
		}

		return this.commitSplice(val)
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

	// Reset the text body to match the server values
	public resyncBody() {
		this.inputBody = this.body
		this.view.replaceText(this.body)
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
			val = [...v]
		let start: number

		// Find first differing character
		for (let i = 0; i < old.length; i++) {
			if (old[i] !== val[i]) {
				start = i
				break
			}
		}

		// New string is appended to the end
		if (start === undefined) {
			start = old.length
		}

		// Right now we simply resend the entire corrected string, including the
		// common part, because I can't figure out a diff algorithm that covers
		// all cases. The backend technically supports the latter.
		const end = val.slice(start).join("")
		this.send(message.splice, {
			start,
			len: -1,
			text: end,
		})
		this.inputBody = old.slice(0, start).join("") + end
	}

	// Close the form and revert to regular post
	public commitClose() {
		this.abandon()
		this.send(message.closePost, null)
	}

	// Turn post form into a regular post, because it has expired after a
	// period of posting ability loss
	public abandon() {
		// Normalize state. The editing attribute remains true, which will cause
		// a close message from the server to close the post one more time and
		// re-render its contents.
		this.body = this.inputBody
		this.view.cleanUp()
		this.closePost()
	}

	// Add a link to the target post in the input
	public addReference(id: number, sel: string) {
		let s = ""
		const body = this.inputBody,
			previousIsLink = />>\d+ *$/.test(body)

		// If already linking a post, put the new one on the next line
		if (!previousIsLink && body && body[body.length - 1] !== " ") {
			s += " "
		}

		// Don't duplicate links, if quoting same post multiple times in
		// succession
		if (id !== this.lasLinked) {
			s += (previousIsLink ? "" : ">") + `>${id} `
		}
		this.lasLinked = id

		if (sel) {
			s += "\n"
			for (let line of sel.split("\n")) {
				s += ">" + line + "\n"
			}
		}

		this.view.replaceText(body + s)
	}

	// Request allocation of a draft post to the server
	private requestAlloc(body: string | null, image: FileData | null) {
		this.sentAllocRequest = true
		const req = newAllocRequest()

		if (body) {
			req["body"] = body
			this.body = body
			this.inputBody = body
		}
		if (image) {
			req["image"] = image
		}

		send(message.insertPost, req)
		handlers[message.postID] = (id: number) => {
			this.setID(id)
			delete handlers[message.postID]
		}
	}

	// Set post ID and add to the post collection
	private setID(id: number) {
		this.id = id
		postSM.feed(postEvent.alloc)
		posts.add(this)
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
		storeMine(data.id)
		if (data.image) {
			this.insertImage(this.image)
		}
	}

	// Upload the file and request its allocation
	public async uploadFile(file?: File) {
		// Already have image
		if (this.image) {
			return
		}

		write(() =>
			this.view.cancel.remove())

		const data = await this.view.upload.uploadFile(file)

		// Upload failed or image added while thumbnailing
		if (!data || this.image) {
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
}
