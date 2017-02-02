import { Model } from '../base'
import { extend } from '../util'
import Collection from './collection'
import PostView from './view'
import { SpliceResponse } from '../client'
import { mine, seenReplies } from "../state"
import { notifyAboutReply } from "../ui"
import { PostData, TextState, PostLink, Command, ImageData } from "../common"

// Generic post model
export class Post extends Model implements PostData {
	collection: Collection
	view: PostView

	// PostData properties
	public editing: boolean
	public deleted: boolean
	public banned: boolean
	public image: ImageData
	public time: number
	public body: string
	public name: string
	public trip: string
	public auth: string
	public state: TextState
	public backlinks: PostLink[]
	public commands: Command[]
	public links: PostLink[]

	constructor(attrs: PostData) {
		super()
		extend(this, attrs)
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
		const char = String.fromCodePoint(code),
			{ view} = this
		this.body += char


		// It is possible to receive text body updates after a post closes,
		// due to server-side buffering optimizations. If so, rerender the body.
		if (char === "\n" || endsWithTag(this.body) || !this.editing) {
			view.reparseBody()
		} else {
			view.appendString(char)
		}
	}

	// Backspace one character in the current line
	public backspace() {
		const needReparse = endsWithTag(this.body) || !this.editing
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
		{start, len, text}: SpliceResponse,
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
			// In case there are multiple links to the same post
			if (!seenReplies.has(this.id)) {
				notifyAboutReply(this)
			}
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
		this.view.renderImage(false, true)
		this.view.autoExpandImage()
	}

	// Spoiler an already allocated imageThreadData
	public spoilerImage() {
		this.image.spoiler = true
		this.view.renderImage(false, true)
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
	return body.endsWith(">") || body.endsWith("**")
}
