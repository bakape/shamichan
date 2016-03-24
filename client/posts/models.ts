import Model from '../model'

export type PostLink = {
	board: string
	op: number
}

export type PostLinks = {[id: number]: PostLink}

export interface PostData {
	editing?: boolean
	deleted?: boolean
	imgDeleted?: boolean
	Image?: ImageData
	op: number
	id: number
	time: number
	board: string
	body: string
	name?: string
	trip?: string
	auth?: string
	email?: string
	backlinks?: PostLinks
	links?: PostLinks
}

export interface ThreadData extends PostData {
	locked?: boolean
	archived?: boolean
	sticky?: boolean
	postCtr?: number
	imageCtr: number
	bumpTime: number
	replyTime: number
}

export interface ImageData {
	apng?: boolean
	audio?: boolean
	spoiler?: number
	fileType: fileTypes
	length?: number
	dims: number[]
	file: string
	size: number
	MD5: string
	SHA1: string
	imgnm: string
	[index: string]: string|number|number[]|boolean
}

export const enum fileTypes {jpeg, png, gif, webm, pdf, svg, mp4, mp3, ogg}

// Generic post model. OP or Reply.
export class Post extends Model {
	constructor(attrs: {[key:string]: any} = {}) {
		super(attrs)
	}
}
