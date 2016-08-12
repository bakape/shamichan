import {default as Model, ModelAttrs} from '../model'
import {extend} from '../util'
import Collection from './collection'
import PostView from './view'

// Generic link object containing target post board and thread
export type PostLink = {
	board: string
	op: number
}

// Map of target to post numbers to their parenthood data
export type PostLinks = {[id: number]: PostLink}

// Data of any post. In addition to server-sent JSON includes the state
// property.
export interface PostData {
	editing: boolean
	image?: ImageData
	time: number
	id: number
	body: string
	name?: string
	trip?: string
	auth?: string
	email?: string
	state: TextState
	backlinks?: PostLinks
	links?: PostLinks
}

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing
export type TextState = {
	spoiler: boolean
	quote: boolean
	line?: string
}

// Data of an OP post
export interface ThreadData extends PostData {
	locked?: boolean
	archived?: boolean
	sticky?: boolean
	postCtr: number
	imageCtr: number
	logCtr: number
	bumpTime: number
	replyTime: number
	subject: string
	board: string
}

// Image data embedable in posts and thread hashes
export type ImageData = {
	apng?: boolean
	audio?: boolean
	spoiler?: boolean
	fileType: fileTypes
	length?: number
	size: number
	dims: number[]
	MD5: string
	SHA1: string
	name: string
	[index: string]: any
}

// Possible file types of a post image
export enum fileTypes {jpg, png, gif, webm, pdf, svg, mp4, mp3, ogg}

// Generic post model
export class Post<V extends PostView<any>> extends Model implements PostData {
	collection: Collection<Post<V>>
	view: V

	// PostData properties
	editing: boolean
	op: number
	image: ImageData
	time: number
	body: string
	name: string
	trip: string
	auth: string
	email: string
	state: TextState
	backlinks: PostLinks
	links: PostLinks

	constructor(attrs: PostData) {
		super()
		extend(this, attrs)
	}

	// Remove the model from its collection, detach all references and allow to
	// be garbage collected.
	remove() {
		if (this.collection) {
			this.collection.remove(this)
		}
		if (this.view) {
			this.view.remove()
		}
	}
}
