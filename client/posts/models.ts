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
	commands?: Command[]
}

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing
export type TextState = {
	spoiler: boolean
	quote: boolean
	iDice: number // Index of the next dice array item to use
	line?: string
}

// Types of hash command entries
export const enum commandType {dice, flip, eightBall, syncWatch, pyu}

// Single hash command result delivered from the server
export type Command = {
	type: commandType
	val: number[]|boolean|string
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
	posts: {[id: number]: PostData}
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
export class Post extends Model implements PostData {
	collection: Collection
	view: PostView

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

	// Append a character to the text body
	append(code: number) {
		const char = String.fromCharCode(code),
			{state} = this

		// Render quote or spoiler tags
		if (state.line === "" && char === ">") {
			state.quote = true
			this.view.startQuote()
		} else if (isQoute(state.line, char)) {
			state.spoiler = !state.spoiler
			this.view.insertSpoilerTag()
		} else {
			this.view.appendString(char)
		}

		state.line += char
	}
}

// Detects if the "**" qoute command is used
const isQoute = (line: string, char: string): boolean =>
	char === "*" && line[line.length - 1] === "*"
