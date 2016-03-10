import Model from '../model'

// Generic post model. OP or Reply.
export class Post extends Model {
	constructor(attrs: {[key:string]: any} = {}) {
		super(attrs)
	}
}
