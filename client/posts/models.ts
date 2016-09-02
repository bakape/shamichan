import Model from '../model'
import {extend} from '../util'
import Collection from './collection'
import PostView from './view'
import {SpliceResponse} from '../client'

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
export interface Command {
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
	posts?: {[id: number]: PostData}
}

// Image data embedable in posts and thread hashes
export interface ImageData {
	apng?: boolean
	audio?: boolean
	spoiler?: boolean
	large?: boolean // Added at runtime to render larger thumbnails
	expanded?: boolean
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
	image: ImageData
	time: number
	body: string
	name: string
	trip: string
	auth: string
	email: string
	state: TextState
	backlinks: PostLinks
	commands: Command[]
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
			{state, view} = this
		this.body += char
		state.line += char

		if (char === "\n") { // Start new line
			view.startNewLine()
			this.state = {
				quote: false,
				spoiler: false,
				iDice: 0,
				line: "",
			}
		} else if (state.line === ">") { // Start qoute
			view.reparseLine()
		} else if (state.line.endsWith("**")) { // Start or close spoiler
			this.resetState()
			view.reparseLine()
		} else {
			view.appendString(char)
		}
	}

	// Reset spoiler and qoute state of the line
	resetState() {
		this.state.spoiler = this.state.quote = false
	}

	// Backspace one character in the current line
	backspace() {
		const {state, view} = this,
			needReparse = state.line === ">" || state.line.endsWith("**")
		state.line = state.line.slice(0, -1)
		this.body = this.body.slice(0, -1)
		if (needReparse) {
			this.resetState()
			view.reparseLine()
		} else {
			view.backspace()
		}
	}

	// Splice the current open line of text
	splice(msg: SpliceResponse) {
		const {state} = this
		state.line = this.spliceLine(state.line, msg)
		this.resetState()
		this.view.reparseLine()
	}

	// Extra method for code reuse in PostForms
	spliceLine(line: string, {start, len, text}: SpliceResponse): string {
		const keep = line.slice(0, start)
		let end: string
		if (len === -1) { // Special meaning - replace till line end
			end = text
		} else {
			end = text + line.slice(start + 1 + len)
		}
		line = keep + end

		// Replace last line in text body
		this.body = this.body.split("\n").slice(0, -1).join("\n") + line

		return line
	}

	// Extend a field on the model, if it exists. Assign if it doesn't
	extendField(key: string, obj: {}) {
		if (this[key]) {
			extend(this[key], obj)
		} else {
			this[key] = obj
		}
	}

	// Insert data about a link to another post into the model
	insertLink(links: PostLinks) {

		// TODO: Trigger Desktop Notification and highlight post, if linking to
		// my post

		this.extendField("links", links)
	}

	// Insert data about another post linking this post into the model
	insertBacklink(links: PostLinks) {
		this.extendField("backlinks", links)
		this.view.renderBacklinks()
	}

	// Insert a new command result into the model
	insertCommand(comm: Command) {
		if (!this.commands) {
			this.commands = [comm]
		}
		this.commands.push(comm)
	}

	// Insert an image into an existing post
	insertImage(img: ImageData) {
		this.image = img

		// TODO: Automatic expansion, if set

		this.view.renderImage()
	}

	// Close an open post and reparse its last line
	closePost() {
		// Posts may be closed from multiple sources. It may be the user
		// closing the post manually, the sheduled cleanup task closing or
		// the check done when writing to open posts. Therefore some
		// duplication is possible. Ignore closing of already closed posts.
		if (!this.editing) {
			return
		}
		this.editing = false
		this.resetState()
		this.view.closePost()
		this.state = null
	}
}

// Model of the opening post of a thread
export class OP extends Post implements ThreadData {
	locked: boolean
	archived: boolean
	sticky: boolean
	postCtr: number
	imageCtr: number
	logCtr: number
	bumpTime: number
	replyTime: number
	subject: string
	board: string

	constructor(data: ThreadData) {
		super(data)
	}
}
