// Common types and constants in a separate module to avoid circular
// dependencies

// Detect FireFox, so we can disable any functionality it's retarded bugs and
// data races break
export const isCuck = navigator.userAgent.toLowerCase().includes("firefox")

// Generic link object containing target post board and thread
export type PostLink = {
	id: number
	op: number
	board: string
}

export const enum ModerationAction {
	banPost,
	unbanPost,
	deletePost,
	deleteImage,
	spoilerImage,
	lockThread,
	deleteBoard,
	meidoVision,
	purgePost,
}

// Contains fields of a post moderation log entry
export interface ModerationEntry {
	type: ModerationAction
	length: number
	by: string
	data: string
}

// Data of any post. In addition to server-sent JSON includes the state
// property.
export interface PostData {
	editing: boolean
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
	moderation?: ModerationEntry[]
}

// State of a post's text. Used for adding enclosing tags to the HTML while
// parsing.
export type TextState = {
	spoiler: boolean
	quote: boolean
	code: boolean
	bold: boolean
	italic: boolean
	red: boolean
	blue: boolean
	haveSyncwatch: boolean
	successive_newlines: number
	iDice: number // Index of the next dice array item to use
}

// Types of hash command entries
export const enum commandType {
	dice, flip, eightBall, syncWatch, pyu, pcount, roulette, rcount,
}

// Single hash command result delivered from the server
export interface Command {
	type: commandType
	val: any
}

// Data of an OP post
export interface ThreadData extends PostData {
	postCtr: number
	imageCtr: number
	replyTime: number
	bumpTime: number
	subject: string
	board: string
	posts?: PostData[]
}

// Data of a board page
export type BoardData = {
	pages: number
	threads: ThreadData[]
}

// Image data embeddable in posts and thread hashes
export interface ImageData {
	audio: boolean
	video: boolean
	spoiler: boolean
	file_type: fileTypes
	thumb_type: fileTypes
	length?: number
	artist?: string
	title?: string
	size: number
	// [width, height, thumbnail_width, thumbnail_height]
	dims: [number, number, number, number]
	md5: string
	sha1: string
	name: string

	// Added client-side
	expanded: boolean           // Thumbnail is expanded
	tallerThanViewport: boolean // Image is taller than the current viewport
	revealed: boolean           // Revealing a hidden image with [Show]
}

// Possible file types of a post image
export enum fileTypes {
	jpg, png, gif, webm, pdf, svg, mp4, mp3, ogg, zip, "7z", "tar.gz", "tar.xz",
	flac, noFile, txt, webp, rar, cbz, cbr,
}

// Return, if source file type can be expanded
export function isExpandable(t: fileTypes): boolean {
	switch (t) {
		case fileTypes.pdf: // Nothing to preview for these
		case fileTypes.mp3:
		case fileTypes.flac:
		case fileTypes.zip:
		case fileTypes["7z"]:
		case fileTypes["tar.gz"]:
		case fileTypes["tar.xz"]:
		case fileTypes.txt:
		case fileTypes.rar:
		case fileTypes.cbr:
		case fileTypes.cbz:
			return false;
		default:
			return true;
	}
}
