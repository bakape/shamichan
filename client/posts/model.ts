import { Model } from '../base'
import { extend } from '../util'
import Collection from './collection'
import PostView from './view'
import { SpliceResponse } from '../client'
import { mine, seenPosts, storeSeenPost, posts, hidden } from "../state"
import { notifyAboutReply } from "../ui"
import {
	PostData, TextState, PostLink, Command, ImageData,
	ModerationEntry, ModerationAction
} from "../common"
import { hideRecursively } from "./hide"
import options from "../options"

// Generic post model
export class Post extends Model implements PostData {
	public collection: Collection
	public view: PostView

	public op: number
	public editing: boolean
	public sage: boolean
	public sticky: boolean
	public locked: boolean
	public seenOnce: boolean
	public hidden: boolean
	public image: ImageData
	public time: number
	public body: string
	public name: string
	public trip: string
	public auth: string
	public subject: string
	public board: string
	public flag: string
	public state: TextState
	public commands: Command[]
	public backlinks: {
		[id: number]: {
			op: number
			board: string
		}
	}
	public links: PostLink[]
	public moderation: ModerationEntry[]

	constructor(attrs: PostData) {
		super()
		extend(this, attrs)
		this.seenOnce = seenPosts.has(this.id)

		// All kinds of interesting races can happen, so best ensure a model
		// always has the state object defined
		this.state = {
			spoiler: false,
			quote: false,
			code: false,
			bold: false,
			italic: false,
			red: false,
			blue: false,
			haveSyncwatch: false,
			successive_newlines: 0,
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

	// Stop post from displaying
	public hide() {
		this.hidden = true
		this.view.hide()
	}

	// Stop hiding the post, if it was hidden
	public unhide() {
		if (!this.hidden) {
			return
		}
		this.hidden = false
		this.view.unhide()
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
		this.spliceText(msg)
		this.view.reparseBody()
	}

	// Extra method for code reuse in post forms
	protected spliceText({ start, len, text }: SpliceResponse) {
		// Must use arrays of chars to properly splice multibyte unicode
		const arr = [...this.body]
		arr.splice(start, len, ...text)
		this.body = arr.join("")
	}

	// Check if this post replied to one of the user's posts and trigger
	// handlers.
	// Set and render backlinks on any linked posts.
	public propagateLinks() {
		if (this.isReply()) {
			notifyAboutReply(this)
		}
		if (this.links) {
			for (let { id } of this.links) {
				const post = posts.get(id)
				if (post) {
					post.insertBacklink({
						id: this.id,
						op: this.op,
						board: this.board,
					})
				}
				if (options.hideRecursively && hidden.has(id)) {
					hideRecursively(this)
				}
			}
		}
	}

	// Returns, if post is a reply to one of the user's posts
	public isReply() {
		if (!this.links) {
			return false
		}
		for (let { id } of this.links) {
			if (mine.has(id)) {
				return true
			}
		}
		return false
	}

	// Insert data about another post linking this post into the model
	public insertBacklink({ id, op, board }: PostLink) {
		if (!this.backlinks) {
			this.backlinks = {}
		}
		this.backlinks[id] = { op, board }
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

	public applyModeration(entry: ModerationEntry) {
		if (!this.moderation) {
			this.moderation = [];
		}
		this.moderation.push(entry);

		const { type, data } = entry;
		switch (type) {
			case ModerationAction.deletePost:
				this.view.el.classList.add("deleted");
				break;
			case ModerationAction.deleteImage:
				if (this.image) {
					this.image = null;
					this.view.removeImage();
				}
				break;
			case ModerationAction.spoilerImage:
				if (this.image) {
					this.image.spoiler = true;
					this.view.renderImage(false);
				}
				break;
			case ModerationAction.lockThread:
				this.locked = data === 'true';
				break;
			case ModerationAction.purgePost:
				if (this.image) {
					this.image = null;
					this.view.removeImage();
				}
				this.body = "";
				this.view.reparseBody()
				break;
		}

		this.view.renderModerationLog()
	}

	public isDeleted(): boolean {
		if (!this.moderation) {
			return false;
		}
		for (let { type } of this.moderation) {
			if (type === ModerationAction.banPost) {
				return true;
			}
		}
		return false;
	}

	public removeImage() {
		this.image = null
		this.view.removeImage()
	}

	// Returns, if this post has been seen already
	public seen() {
		if (this.hidden || this.seenOnce) {
			return true
		}
		if (document.hidden) {
			return false
		}
		if (this.seenOnce = this.view.scrolledPast()) {
			storeSeenPost(this.id, this.op)
		}
		return this.seenOnce
	}
}

function endsWithTag(body: string): boolean {
	const sl = body[body.length - 2]
	switch (body[body.length - 1]) {
		case ">":
			return true
		case "*":
			return sl === "*"
		case "`":
			return sl === "`"
		case "@":
			return sl === "@"
		case "~":
			return sl === "~"
		case "r":
			return sl === "^"
		case "b":
			return sl === "^"
	}
	return false
}
