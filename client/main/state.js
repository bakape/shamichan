/*
 * Central model keeping the state of the page
 */

import {_, Backbone} from 'main'
main.state = module.exports

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
export const page = new Backbone.Model(read(location.href))

// Hot-reloadable configuration

// TODO: We need actual listeners to this model for hot reloads

// Tracks the synchronisation counter of each thread
export let syncs = {}

// Posts I made in this tab
export const ownPosts = {}

// remember which posts are mine for two days
export const mine = new main.Memory('mine', 2, true)

// All posts currently displayed
export const posts = new Backbone.Collection()

main.on('state:clear', () => {
	/*
	 * Emptying the whole element should be faster than removing each post
	 * individually through models and listeners
	 */
	main.$threads.innerHTML = ''

	// The <threads> tag has already been emptied, no need to perform
	// element removal with the default `.remove()` method
	models.each(model =>
		model.dispatch('stopListening'))

	posts.reset()

	// Prevent old threads from syncing
	exports.syncs = {}
	main.request('massExpander:unset')
})

// Post links verified server-side
export const links = {}

export function addLinks(addition) {
	if (addition) {
		_.extend(links, addition);
	}
}
