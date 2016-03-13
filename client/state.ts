/*
 Stores the state of the web application
*/

import Model from './model'
import {Post} from './posts/models'
import Collection from './collection'
import {getID} from './util'

// Allows us to typecheck configs. See config/defaults.json for more info.
type Configs = {
	boards: {
		enabled: string[]
		boards: {[name: string]: {title: string}}
		default: string
		staff: string
		psuedo: string[][]
		links: string[][]
	}

	lang: {
		default: string
		enabled: string[]
	}

	staff: {
		classes: {[name: string]: StaffClass}
		keyword: string
	}

	images: {
		thumb: {
			thumbDims: number[]
			midDims: number[]
		}
		spoilers: number[]
		hats: boolean
	}

	banners: string[]
	FAQ: string[]
	eightball: string[]
	radio: boolean
	illyaDance: boolean
	feedbackEmail: string
	defaultCSS: string
	infoBanner: string
}

type StaffClass = {
	alias: string
	rights: {[right: string]: boolean}
}

// Configuration passed from the server. Some values can be changed during
// runtime.
export const config: Configs = (window as any).config

// Indicates, if in mobile mode. Determined server-side.
export const isMobile: boolean = (window as any).isMobile

interface PageState {
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
	}
}

// Load initial page state
export const page = new Model(read(location.href))

// Cached DOM elements
export const $thread = document.query('threads')
export const $name = document.query('#name')
export const $email = document.query('#email')
export const $banner = document.query('#banner')
export const $loading = document.query('#loadingImage')

// All posts currently displayed
export const posts = new Collection<Post>()

// Posts I made in this tab
export const ownPosts = new Set<number>()



// Tracks the synchronisation counter of each thread/board
export const syncs = {}

// Retrieve model of closest parent post
export function getModel(el: Element): Post {
	const id = getID(el)
	if (!id) {
		return null
	}
	return posts.get(id)
}

// Display or hide the loading animation
export function displayLoading(loading: boolean) {
	$loading.style.display = loading ? 'block' : 'none'
}
