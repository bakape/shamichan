import { Model } from '../base'
import { extend } from '../util'
import Collection from './collection'
import PostView from './view'
import { SpliceResponse } from '../client'
import { mine } from "../state"
import { notifyAboutReply } from "../ui"
import { PostData, TextState, PostLink, Command, ImageData } from "../common"

// Generic post model
export class Post extends Model implements PostData {
	collection: Collection
	view: PostView

	// PostData properties
	public op: number
	public editing: boolean
	public deleted: boolean
	public banned: boolean
	public image: ImageData
	public time: number
	public body: string
	public name: string
	public trip: string
	public auth: string
	public subject: string
	public board: string
	public state: TextState
	public backlinks: PostLink[]
	public commands: Command[]
	public links: PostLink[]

	constructor(attrs: PostData) {
		super()
		extend(this, attrs)

		// All kinds of interesting races can happen, so best ensure a model
		// always has the state object defined
		this.state = {
			spoiler: false,
			quote: false,
			lastLineEmpty: false,
			code: false,
			haveSyncwatch: false,
			iDice: 0,
		}
	}

	// Remove the model from its collection, detach all references and allow to
	// be garbage collected.
	public remove() {
		if (this.collection) {
			this.collection.remove(this)
		}
		if (this.view) {
			this.view.remove()
		}
	}

	// Append a character to the text body
	public append(code: number) {
		const char = String.fromCodePoint(code)
		this.body += char

		// It is possible to receive text body updates after a post closes,
		// due to server-side buffering optimizations. If so, rerender the body.
		const needReparse = char === "\n"
			|| !this.editing
			|| this.state.code
			|| endsWithTag(this.body)
		if (needReparse) {
			this.view.reparseBody()
		} else {
			this.view.appendString(char)
		}
	}

	// Backspace one character in the current line
	public backspace() {
		const needReparse = this.body[this.body.length - 1] === "\n"
			|| !this.editing
			|| this.state.code
			|| endsWithTag(this.body)
		this.body = this.body.slice(0, -1)
		if (needReparse) {
			this.view.reparseBody()
		} else {
			this.view.backspace()
		}
	}

	// Splice the current open line of text
	public splice(msg: SpliceResponse) {
		this.body = this.spliceText(this.body, msg)
		this.view.reparseBody()
	}

	// Extra method for code reuse in post forms
	protected spliceText(
		body: string,
		{ start, len, text }: SpliceResponse,
	): string {
		// Must use arrays of chars to properly splice multibyte unicode
		const keep = Array.from(body).slice(0, start),
			t = Array.from(text)
		let end: string[]
		if (len === -1) { // Special meaning - replace till line end
			end = t
		} else {
			end = t.concat(Array.from(body).slice(start + 1 + len))
		}
		body = keep.concat(end).join("")

		// Replace last line in text body
		const iLast = this.body.lastIndexOf("\n")
		this.body = this.body.substring(0, iLast + 1) + body

		return body
	}

	// Check if this post replied to one of the user's posts and trigger
	// handlers
	public checkRepliedToMe() {
		for (let [id] of this.links) {
			if (!mine.has(id)) {
				continue
			}
			notifyAboutReply(this)
		}
	}

	// Insert data about another post linking this post into the model
	public insertBacklink(id: number, op: number) {
		const l: [number, number] = [id, op]
		if (this.backlinks) {
			this.backlinks.push(l)
		} else {
			this.backlinks = [l]
		}
		this.view.renderBacklinks()
	}

	// Insert an image into an existing post
	public insertImage(img: ImageData) {
		this.image = img
		this.view.renderImage(false)
		this.view.autoExpandImage()
	}

	// Spoiler an already allocated imageThreadData
	public spoilerImage() {
		this.image.spoiler = true
		this.view.renderImage(false)
	}

	// Close an open post and reparse its last line
	public closePost() {
		this.editing = false
		this.view.closePost()
	}

	// Set post as banned
	public setBanned() {
		if (this.banned) {
			return
		}
		this.banned = true
		this.view.renderBanned()
	}
}

function endsWithTag(body: string): boolean {
	switch (body[body.length - 1]) {
		case ">":
			return true
		case "*":
			return body[body.length - 2] === "*"
		case "`":
			return body[body.length - 2] === "`"
	}
	return false
}
