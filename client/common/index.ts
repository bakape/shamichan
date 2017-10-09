// Common types and constants in a separate module to avoid circular
// dependencies

// Detect FireFox, so we can disable any functionality it's retarded bugs and
// data races break
export const isCuck = navigator.userAgent.toLowerCase().includes("firefox")

// Generic link object containing target post board and thread
export type PostLink = [number, number]

// Data of any post. In addition to server-sent JSON includes the state
// property.
export interface PostData {
	editing: boolean
	deleted: boolean
	banned: boolean
	sage: boolean
	sticky: boolean
	locked: boolean
	image?: ImageData
	time: number
	id: number
	op: number
	body: string
	name: string
	trip: string
	auth: string
	board?: string
	flag?: string
	state: TextState
	links?: PostLink[]
	commands?: Command[]
}

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing.
export type TextState = {
	spoiler: boolean
	quote: boolean
	code: boolean
	haveSyncwatch: boolean
	newlines: number
	iDice: number // Index of the next dice array item to use
}

// Types of hash command entries
const enum commandType { dice, flip, eightBall, syncWatch, pyu, pcount }

// Single hash command result delivered from the server
export interface Command {
	type: commandType
	val: any
}

// Data of an OP post
export interface ThreadData extends PostData {
	nonLive: boolean
	postCtr: number
	imageCtr: number
	replyTime: number
	bumpTime: number
	subject: string
	board: string
	posts?: PostData[]
}

// Image data embeddable in posts and thread hashes
export interface ImageData {
	apng: boolean
	audio: boolean
	video: boolean
	spoiler: boolean
	fileType: fileTypes
	thumbType: fileTypes
	length?: number
	artist?: string
	title?: string
	size: number
	// [width, height, thumbnail_width, thumbnail_height]
	dims: [number, number, number, number]
	MD5: string
	SHA1: string
	name: string

	// Added client-side
	large: boolean              // Render larger thumbnails
	expanded: boolean           // Thumbnail is expanded
	tallerThanViewport: boolean // Image is taller than the current viewport
	revealed: boolean           // Revealing a hidden image with [Show]
}

// Possible file types of a post image
export enum fileTypes {
	jpg, png, gif, webm, pdf, svg, mp4, mp3, ogg, zip, "7z", "tar.gz", "tar.xz",
	flac, noFile, txt
}
