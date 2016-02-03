/*
 * Central model keeping the state of the page
 */

import {extend} from 'underscore'
import Memory from './memory'
import {randomID, getID} from './util'
import Model from './model'
import Collection from './collection'

// Read page state by parsing a URL
export function read(href) {
	const state = {
		board: href.match(/\/([a-zA-Z0-9]+?)\//)[1],
		thread: href.match(/\/(\d+)(:?#\d+)?(?:[\?&]\w+=\w+)*$/),
		// Displayed last N posts setting on thread pages
		lastN: href.match(/[\?&]last=(\d+)/)
	}
	for (let key of ['thread', 'lastN']) {
		const val = state[key]
		state[key] = val ? parseInt(val[1]) : 0
	}
	return state
}

// Initial page state
const initial = read(location.href)
initial.tabID = randomID(32)
export let page = new Model(initial)

// Hot-reloadable configuration

// TODO: We need actual listeners to this model for hot reloads

// Tracks the synchronisation counter of each thread
export let syncs = {}

// Posts I made in this tab
export const ownPosts = {}

// Configuration object, passed from the server
export const config = window.config

// Hash of the the configuration object
export const configHash = window.configHash

// Indicates, if in mobile mode. Determined server-side.
export const isMobile = window.isMobile

// Cached DOM elements
export const $thread = document.query('threads')
export const $name = document.query('#name')
export const $email = document.query('#email')
export const $banner = document.query('#banner')

// Remember which posts are mine for two days
export const mine = new Memory('mine', 2)

// All posts currently displayed
export const posts = new Collection()

// Clear the current post state and HTML
export function clear() {
	/*
	 * Emptying the whole element should be faster than removing each post
	 * individually through models and listeners
	 */
	$threads.innerHTML = ''

	// The <threads> tag has already been emptied, no need to perform
	// element removal with the default `.remove()` method
	models.each(model =>
		model.dispatch('stopListening'))

	posts.reset()

	// Prevent old threads from syncing
	exports.syncs = {}
	events.request('massExpander:unset')
}

// Retrieve model of closest parent post
export function getModel(el) {
	const id = getID(el)
	if (!id) {
		return null
	}
	return posts.get(id)
}
