/*
 * Central model keeping the state of the page
 */

// Configuration object, passed from the server
export const config = window.config

// Hash of the the configuration object
export const configHash = window.configHash

// Remember which posts are mine for two days
export const mine = new Memory('mine', 2)

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
