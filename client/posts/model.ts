import { Model } from '../base'
import { extend } from '../util'
import Collection from './collection'
import PostView from './view'
import { mine, seenPosts, storeSeenPost, posts, hidden } from "../state"
import { notifyAboutReply } from "../ui"
import { PostData, TextState, PostLink, Command, ImageData } from "../common"
import { hideRecursively } from "./hide"

// Generic post model
export class Post extends Model implements PostData {
	public collection: Collection
	public view: PostView

	public op: number
	public deleted: boolean
	public sage: boolean
	public banned: boolean
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
	public posterID: string
	public state: TextState
	public commands: Command[]
	public backlinks: { [id: number]: number }
	public links: PostLink[]

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

	// Check if this post replied to one of the user's posts and trigger
	// handlers.
	// Set and render backlinks on any linked posts.
	public propagateLinks() {
		if (this.isReply()) {
			notifyAboutReply(this)
		}
		if (this.links) {
			for (let [id] of this.links) {
				const post = posts.get(id)
				if (post) {
					post.insertBacklink(this.id, this.op)
				}
				if (hidden.has(id)) {
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
		for (let [id] of this.links) {
			if (mine.has(id)) {
				return true
			}
		}
		return false
	}

	// Insert data about another post linking this post into the model
	public insertBacklink(id: number, op: number) {
		if (!this.backlinks) {
			this.backlinks = {}
		}
		this.backlinks[id] = op
		this.view.renderBacklinks()
	}

	// Spoiler an already allocated imageThreadData
	public spoilerImage() {
		this.image.spoiler = true
		this.view.renderImage(false)
	}

	// Set post as banned
	public setBanned() {
		if (this.banned) {
			return
		}
		this.banned = true
		this.view.renderBanned()
	}

	// Set post as deleted
	public setDeleted() {
		this.deleted = true
		this.view.renderDeleted()
	}

	public removeImage() {
		this.image = null
		this.view.removeImage()
	}

	// Returns, if this post has been seen already
	public seen() {
		if (this.seenOnce) {
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
