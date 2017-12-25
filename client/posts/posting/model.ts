import { message, send, handlers } from "../../connection"
import { Post } from "../model"
import FormView from "./view"
import { storeMine, page, storeSeenPost } from "../../state"
import { newAllocRequest } from "./identity"

// Form Model of an OP post
export default class FormModel extends Post {
	// Need to solve a captcha to submit
	public needCaptcha: boolean

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
		const old = this.body

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

		this.body = val
	}

	// Add a link to the target post in the input
	public addReference(id: number, sel: string) {
		let s = ""
		const old = this.body,
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

	// Commit a post to server
	public async commit() {
		let file: File
		if (this.view.upload) {
			file = this.view.upload.input.files[0]
		}
		if (!file && !this.body) {
			// Empty post
			return this.remove()
		}

		const req = newAllocRequest()
		if (file) {
			req["image"] = await this.view.upload.uploadFile(file)
		}
		req["body"] = this.body

		send(message.insertPost, req)
		handlers[message.postID] = (id: number) => {
			storeSeenPost(this.id, this.op)
			storeMine(this.id, this.op)
			delete handlers[message.postID]
		}
		this.remove()
	}

	// Upload the files to be uploaded
	public setUploads(files: FileList) {
		if (files && this.view.upload) {
			(this.view.upload.input.files as any) = files
		}
	}
}
