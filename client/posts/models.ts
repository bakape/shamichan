import Model from '../model'

export type PostLink = {
	board: string
	op: number
}

export type PostLinks = {[id: number]: PostLink}

// Generic post model. OP or Reply.
export class Post extends Model {
	constructor(attrs: {[key:string]: any} = {}) {
		super(attrs)
	}
}
