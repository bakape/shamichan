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
	image?: ImageData
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
	state?: number[]
	largeThumb?: boolean
	locked?: boolean
	archived?: boolean
	sticky?: boolean
	postCtr?: number
	imageCtr: number
	bumpTime: number
	replyTime: number
	subject?: string
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
	constructor() {
		super()
	}
}
