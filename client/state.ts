// Stores the state of the web application

import { Post, PostCollection } from './posts'
import { getClosestID, emitChanges, ChangeEmitter } from './util'
import { readIDs, storeID } from './db'
import { send } from './connection'

// Server-wide global configurations
interface Configs {
	captcha: boolean
	mature: boolean // Website intended for mature audiences
	defaultLang: string
	defaultCSS: string
	captchaPublicKey: string
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

// Replies to this user's posts the user has already seen
export let seenReplies: Set<number>

// Explicitly hidden posts and threads
export let hidden: Set<number>

// Debug mode with more verbose logging
export let debug: boolean = /[\?&]debug=true/.test(location.href)

// Read page state by parsing a URL
export function read(href: string): PageState {
	const [, board, thread] = href
		.match(/\/(\w+)\/(\w+)?(?:\?[^#]+)?(?:#[^#]+)?$/)
	const lastN = href.match(/[\?&]last=(\d+)/)
	const state = {
		href,
		board: decodeURIComponent(board),
		lastN: lastN ? parseInt(lastN[1]) : 0,
	} as PageState

	state.catalog = thread === "catalog"
	if (!state.catalog) {
		state.thread = thread ? parseInt(thread) : 0
	} else {
		state.thread = 0
	}

	return state
}

// Load post number sets from the database
export function loadFromDB(): Promise<Set<number>[]> {
	return Promise.all([
		readIDs("mine").then(ids =>
			mine = new Set(ids)),
		readIDs("seen").then(ids =>
			seenReplies = new Set(ids)),
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
	return posts.get(id)
}

// Display or hide the loading animation
export function displayLoading(display: boolean) {
	loading.style.display = display ? 'block' : 'none'
}

; (window as any).debugMode = () => {
	debug = true;
	(window as any).send = send
}
