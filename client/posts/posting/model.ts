import { message, send, handlers } from "../../connection"
import { Post } from "../model"
import { PostData } from "../../common"
import FormView from "./view"
import { posts, storeMine, page, storeSeenPost } from "../../state"
import { postSM, postEvent } from "."
import { newAllocRequest } from "./identity"

// Form Model of an OP post
export default class FormModel extends Post {
	public needCaptcha: boolean // Need to solve a captcha to allocate

	// Text that is not submitted yet to defer post allocation
	public bufferedFile: File // Same for file uploads

	public inputBody = ""
	public view: FormView

	// Pass and ID, if you wish to hijack an existing model. To create a new
	// model pass zero.
	constructor() {
		// Initialize state
		super({
			id: 0,
			op: page.thread,
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

		// Remove any lines past 100
		const lines = val.split("\n")
		if (lines.length - 1 > 100) {
			const trimmed = lines.slice(0, 100).join("\n")
			this.view.trimInput(val.length - trimmed.length)
			return this.parseInput(trimmed)
		}
	}

	// Turn post form into a regular post, because it has expired after a
	// period of posting ability loss
	public abandon() {
		this.view.cleanUp()
	}

	// Add a link to the target post in the input
	public addReference(id: number, sel: string) {
		let s = ""
		const old = this.inputBody,
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

		this.view.replaceText(old + s)
	}

	// Commit a post made with live updates disabled
	public async commitNonLive() {
		if (!this.bufferedFile) {
			return postSM.feed(postEvent.done)
		}

		const req = newAllocRequest()
		if (this.bufferedFile) {
			req["image"] = await this.view.upload.uploadFile(this.bufferedFile)
		}
		req["body"] = this.body = this.inputBody

		send(message.insertPost, req)
		handlers[message.postID] = (id: number) => {
			this.id = id
			this.op = page.thread
			this.seenOnce = true
			storeSeenPost(this.id, this.op)
			storeMine(this.id, this.op)
			posts.add(this)
			delete handlers[message.postID]
		}
	}

	// Handle draft post allocation
	// TODO
	public onAllocation(data: PostData) {
		// // May sometimes be called multiple times, because of reconnects
		// if (this.isAllocated) {
		// 	return
		// }

		// this.isAllocated = true
		// extend(this, data)
		// this.view.renderAlloc()
		// if (data.image) {
		// 	this.insertImage(this.image)
		// }
		// if (this.nonLive) {
		// 	this.propagateLinks()
		// 	postSM.feed(postEvent.done)
		// }
	}

	// Upload the file and request its allocation
	public uploadFile(file?: File) {
		this.bufferedFile = file || this.view.upload.input.files[0]
	}
}
