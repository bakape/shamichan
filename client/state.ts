// Stores the state of the web application

import {emitChanges, ChangeEmitter} from './model'
import {Post} from './posts/models'
import PostCollection from './posts/collection'
import {getID, fetchBoarConfigs} from './util'
import {db} from './db'
import {write} from './render'
import {send} from './connection'
import PostView from './posts/view'

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
	links: StringMap
}

// Board-specific configurations
export interface BoardConfigs extends ChangeEmitter {
	readOnly: boolean
	textOnly: boolean
	forcedAnon: boolean
	hashCommands: boolean
	spoilers: boolean
	codeTags: boolean
	spoiler: string
	title: string
	notice: string
	banners: string[]
	[index: string]: any
}

interface PageState extends ChangeEmitter {
	board: string
	thread: number
	lastN: number
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
export const posts = new PostCollection<Post<PostView<any>>>()

// Posts I made in any tab
export let mine: Set<number>

// Posts I made in this tab
export const ownPosts = new Set<number>()

// Tracks the synchronisation progress of the current thread/board
export let syncCounter: number

// Debug mode with more verbose logging
export let debug: boolean = /[\?&]debug=true/.test(location.href)

// Read page state by parsing a URL
function read(href: string): PageState {
	const board = href.match(/\/(\w+)\//)[1],
		thread = href.match(/\/(\d+)(:?#\d+)?(?:[\?&]\w+=\w+)*$/),
		lastN = href.match(/[\?&]last=(\d+)/)
	return {
		board: decodeURIComponent(board),
		thread: thread ? parseInt(thread[1]) : 0,
		lastN: lastN ? parseInt(lastN[1]) : 0,
	} as PageState
}

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

// Fetch and load board-specfic configurations
export const loadBoardConfig = async () =>
	boardConfig.replaceWith(await fetchBoarConfigs(page.board))

// Retrieve model of closest parent post
export function getModel(el: Element): Post<PostView<any>> {
	const id = getID(el)
	if (!id) {
		return null
	}
	return posts.get(id)
}

const $loading = document.querySelector('#loadingImage')

// Display or hide the loading animation
export const displayLoading = (loading: boolean) =>
	write(() =>
		$loading.style.display = loading ? 'block' : 'none')

; (window as any).debugMode = () => {
	debug = true
	; (window as any).send = send
}
