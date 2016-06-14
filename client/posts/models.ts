import {default as Model, ModelAttrs} from '../model'
import {extend} from '../util'

export type PostLink = {
	board: string
	op: number
}

export type PostLinks = {[id: number]: PostLink}

export class Post extends Model {
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
}

export type ImageData = {
	apng?: boolean
	audio?: boolean
	spoiler?: number
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
export class Reply extends Post {
	editing: boolean

	constructor(attrs: ModelAttrs) {
		super(attrs)
	}
}

export class Thread extends Post {
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
