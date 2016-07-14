import {default as Model, ModelAttrs} from '../model'
import {extend} from '../util'
import Collection from './collection'
import PostView from './view'

export type PostLink = {
	board: string
	op: number
}

export type PostLinks = {[id: number]: PostLink}

export class Post<V extends PostView<any>> extends Model {
	collection: Collection<Post<V>>
	views: V[] = []

	op: number
	image: ImageData
	time: number
	board: string
	body: string
	name: string
	trip: string
	auth: string
	email: string
	deleted: boolean
	imgDeleted: boolean
	state: number[]
	backlinks: PostLinks
	links: PostLinks

	constructor(attrs: ModelAttrs = {}) {
		super()
		extend(this, attrs)
	}

	// Remove the model from its collection, detach all references and allow to
	// be garbage collected.
	remove() {
		if (this.collection) {
			this.collection.remove(this)
		}
		for (let view of this.views) {
			view.remove()
		}
	}

	// Attach a view to the model. Each model can have several views attached to
	// it.
	attach(view: V) {
		this.views.push(view)
	}

	// Detach a view from the model
	detach(view: V) {
		const {views} = this
		this.views = views.splice(views.indexOf(view), 1)
	}
}

export type ImageData = {
	apng?: boolean
	audio?: boolean
	spoiler?: boolean
	fileType: fileTypes
	length?: number
	dims: number[]
	size: number
	MD5: string
	SHA1: string
	imgnm: string
	[index: string]: string|number|number[]|boolean
}

export enum fileTypes {jpg, png, gif, webm, pdf, svg, mp4, mp3, ogg}

// Generic post model. OP or Reply.
export class Reply extends Post<PostView<any>> {
	editing: boolean

	constructor(attrs: ModelAttrs) {
		super(attrs)
	}
}

export class Thread extends Post<PostView<any>> {
	locked: boolean
	archived: boolean
	sticky: boolean
	postCtr: number
	imageCtr: number
	bumpTime: number
	replyTime: number
	subject: string

	constructor(attrs: ModelAttrs) {
		super(attrs)
	}
}
