/*
 Stores the state of the web application
*/

import Model from './model'
import {Post} from './posts/models'
import Collection from './collection'
import {getID} from './util'

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
