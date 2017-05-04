// Stores the state of the web application

import { Post, PostCollection } from './posts'
import { getClosestID, emitChanges, ChangeEmitter } from './util'
import { readIDs, storeID } from './db'
import { send } from './connection'

// Server-wide global configurations
interface Configs {
	captcha: boolean
	mature: boolean // Website intended for mature audiences
	disableUserBoards: boolean
	pruneThreads: boolean
	threadExpiryMin: number
	threadExpiryMax: number
	defaultLang: string
	defaultCSS: string
	imageRootOverride: string
	links: { [key: string]: string }
}

// Board-specific configurations
export interface BoardConfigs {
	readOnly: boolean
	textOnly: boolean
	forcedAnon: boolean
	title: string
	notice: string
	rules: string
	[index: string]: any
}

// The current state of a board or thread page
export interface PageState extends ChangeEmitter {
	catalog: boolean
	thread: number
	lastN: number
	board: string
	href: string
}

const thirtyDays = 30 * 24 * 60 * 60 * 1000,
	loading = document.getElementById('loading-image')

// Configuration passed from the server. Some values can be changed during
// runtime.
export const config: Configs = (window as any).config

// Currently existing boards
export let boards: string[] = (window as any).boards

export let boardConfig: BoardConfigs

// Load initial page state
export const page = emitChanges<PageState>(read(location.href))

// All posts currently displayed
export const posts = new PostCollection()

// Posts I made in any tab
export let mine: Set<number>

// Posts that the user has already seen or scrolled past
export let seenPosts: Set<number>

// Replies to this user's posts the user has already seen
export let seenReplies: Set<number>

// Explicitly hidden posts and threads
export let hidden: Set<number>

// Debug mode with more verbose logging
export let debug: boolean = /[\?&]debug=true/.test(location.href)

// Read page state by parsing a URL
export function read(href: string): PageState {
	// Convert to absolute URL, if relative
	const a = document.createElement("a")
	a.href = href
	href = a.href

	const u = new URL(href),
		thread = u.pathname.match(/^\/\w+\/(\d+)/)
	return {
		href,
		board: u.pathname.match(/^\/(\w+)\//)[1],
		lastN: /[&\?]last=100/.test(u.search) ? 100 : 0,
		catalog: /^\/\w+\/catalog/.test(u.pathname),
		thread: parseInt(thread && thread[1]) || 0,
	} as PageState
}

// Load post number sets from the database
export function loadFromDB(): Promise<Set<number>[]> {
	return Promise.all([
		readIDs("mine").then(ids =>
			mine = new Set(ids)),
		readIDs("seen").then(ids =>
			seenReplies = new Set(ids)),
		readIDs("seenPost").then(ids =>
			seenPosts = new Set(ids)),
		readIDs("hidden").then((ids) =>
			hidden = new Set(ids)),
	])
}

// Store the ID of a post this client created
export function storeMine(id: number) {
	mine.add(id)
	storeID("mine", id, thirtyDays)
}

// Store the ID of a post that replied to one of the user's posts
export function storeSeenReply(id: number) {
	seenReplies.add(id)
	storeID("seen", id, thirtyDays)
}

export function storeSeenPost(id: number) {
	seenReplies.add(id)
	storeID("seenPost", id, thirtyDays)
}

// Store the ID of a post or thread to hide
export function storeHidden(id: number) {
	hidden.add(id)
	storeID("hidden", id, thirtyDays)
}

export function setBoardConfig(c: BoardConfigs) {
	boardConfig = c
}

// Retrieve model of closest parent post
export function getModel(el: Element): Post {
	const id = getClosestID(el)
	if (!id) {
		return null
	}
	return PostCollection.getFromAll(id)
}

// Display or hide the loading animation
export function displayLoading(display: boolean) {
	loading.style.display = display ? 'block' : 'none'
}

; (window as any).debugMode = () => {
	debug = true;
	(window as any).send = send
}
