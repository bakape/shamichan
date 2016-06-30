/*
 Stores the state of the web application
*/

import {emitChanges} from './model'
import {Post} from './posts/models'
import Collection from './collection'
import {getID} from './util'
import {db} from './db'
import {write} from './render'

// Allows us to typecheck configs. See config/defaults.json for more info.
type Configs = {
	radio: boolean
	hats: boolean
	illyaDance: boolean
	maxSubjectLength: number
	defaultLang: string
	defaultCSS: string
	FAQ: string
	boards: string[]
	langs: string[]
	links: string[][]
	spoilers: number[]
}

type BoardConfigs = {
	staffClasses: string[]
}

// Configuration passed from the server. Some values can be changed during
// runtime.
export const config: Configs = (window as any).config

// Indicates, if in mobile mode. Determined server-side.
export const isMobile: boolean = (window as any).isMobile

interface PageState extends ChangeEmitter {
	board: string
	thread: number
	lastN: number
}

// Read page state by parsing a URL
function read(href: string): PageState {
	const board = href.match(/\/([a-zA-Z0-9]+?)\//)[1],
		thread = href.match(/\/(\d+)(:?#\d+)?(?:[\?&]\w+=\w+)*$/),
		lastN = href.match(/[\?&]last=(\d+)/)
	return {
		board,
		thread: thread ? parseInt(thread[1]) : 0,
		lastN: lastN ? parseInt(lastN[1]) : 0,
	} as PageState
}

// Load initial page state
export const page = emitChanges<PageState>(read(location.href))

// All posts currently displayed
export const posts = new Collection<Post>()

// Posts I made in any tab
export let mine: Set<number>

// Load post number sets from the database
export async function loadFromDB() {
	const resMine = await db
		.transaction('posts', 'readonly')
		.objectStore('posts')
		.get('mine')
		.exec()
	delete resMine.id
	mine = new Set<number>([resMine])
}

// Posts I made in this tab
export const ownPosts = new Set<number>()

// Tracks the synchronisation progress of the current thread/board
export let syncCounter: number

// Retrieve model of closest parent post
export function getModel(el: Element): Post {
	const id = getID(el)
	if (!id) {
		return null
	}
	return posts.get(id)
}

const $loading = document.querySelector('#loadingImage')

// Display or hide the loading animation
export function displayLoading(loading: boolean) {
	write(() => $loading.style.display = loading ? 'block' : 'none')
}

// Debug mode with more verbose logging
export let debug: boolean = /[\?&]debug=true/.test(location.href)

// Allow switching to debug mode from the JS console
; (window as any).debugMode = (setting: boolean) =>
	debug = setting

// ID of the current tab on the server. Set after synchronisation.
export let clientID: string

export function setClientID(id: string) {
	clientID = id
}
