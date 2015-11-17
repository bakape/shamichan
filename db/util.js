/**
 * Various helper utilities
 */

const r = require('rethinkdb'),
    {rcon} = global

/**
 * Shorthand for post retrieval
 * @param {int} num
 * @returns {Object}
 */
export function getPost(num) {
	return r.table('posts').get(num)
}

/**
 * Remove properties the client should not be seeing
 * @param {Object} post
 */
export function formatPost(post) {
	for (let key of ['ip', 'deleted', 'imgDeleted']) {
		delete post[key];
	}
}
