// Stores the state of the web application

import {emitChanges, ChangeEmitter} from './model'
import {Post} from './posts/models'
import PostCollection from './posts/collection'
import {getClosestID} from './util'
import {db} from './db'
import {write} from './render'
import {send} from './connection'

// Server-wide global configurations
interface Configs extends ChangeEmitter {
	radio: boolean
	hats: boolean
	illyaDance: boolean
	captcha: boolean
	defaultLang: string
	defaultCSS: string
	FAQ: string
	captchaPublicKey: string
	boards: string[]
	links: {[key: string]: string}
}

// Board-specific configurations
export interface BoardConfigs extends ChangeEmitter {
	readOnly: boolean
	textOnly: boolean
	forcedAnon: boolean
	hashCommands: boolean
	spoilers: boolean     // Text spoilers
	codeTags: boolean
	spoiler: string       //Image spoiler
	title: string
	notice: string
	rules: string
	banners: string[]
	[index: string]: any
}

// The current state of a board or thread page
export interface PageState extends ChangeEmitter {
	thread: number
	lastN: number
	board: string
	href: string
}

// Configuration passed from the server. Some values can be changed during
// runtime.
export const config: Configs = (window as any).config

// Indicates, if in mobile mode. Determined server-side.
export const isMobile: boolean = (window as any).isMobile

export let boardConfig: BoardConfigs = emitChanges({} as BoardConfigs)

// Load initial page state
export const page = emitChanges<PageState>(read(location.href))

// All posts currently displayed
export const posts = new PostCollection()

// Posts I made in any tab
export let mine: Set<number>

// Tracks the synchronisation progress of the current thread/board
export let syncCounter: number

// Debug mode with more verbose logging
export let debug: boolean = /[\?&]debug=true/.test(location.href)

// Set the synchronisation counter
export function setSyncCounter(ctr: number) {
	syncCounter = ctr
}

// Read page state by parsing a URL
export function read(href: string): PageState {
	const noHash = href.split("#")[0],
		board = noHash.match(/\/(\w+)\//)[1],
		thread = noHash.match(/\/(\d+)(?:[\?&]\w+=\w+)*$/),
		lastN = noHash.match(/[\?&]lastN=(\d+)/)
	return {
		href,
		board: decodeURIComponent(board),
		thread: thread ? parseInt(thread[1]) : 0,
		lastN: lastN ? parseInt(lastN[1]) : 0,
	} as PageState
}

// Load post number sets from the database
export function loadFromDB(): Promise<void> {
	return  Promise.all([loadMine()])
}

// Load post's this client has created
function loadMine(): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		const ids: number[] = []
		const req =
			db
			.transaction("mine", "readonly")
			.objectStore("mine")
			.openCursor()

		req.onerror = err =>
			reject(err)

		req.onsuccess = event => {
			const cursor = (event as any).target.result as IDBCursorWithValue
			if (cursor) {
				ids.push(cursor.value.id)
				cursor.continue()
			} else {
				mine = new Set<number>(ids)
				resolve()
			}
		}
	})
}

// Store the ID of a post this client created
export function storeMine(id: number) {
	mine.add(id)
	const trans = db.transaction("mine", "readwrite")

	trans.onerror = err => {
		throw err
	}

	const req = trans.objectStore("mine").add({
		id,
		expires: Date.now() + 10 * 24 * 60 * 60, // Expire in 10 days
	})

	req.onerror = err => {
		throw err
	}
}

// Retrieve model of closest parent post
export function getModel(el: Element): Post {
	const id = getClosestID(el)
	if (!id) {
		return null
	}
	return posts.get(id)
}

const $loading = document.querySelector('#loading-image') as HTMLElement

// Display or hide the loading animation
export function displayLoading(loading: boolean) {
	write(() =>
		$loading.style.display = loading ? 'block' : 'none')
}

; (window as any).debugMode = () => {
	debug = true
	; (window as any).send = send
}
