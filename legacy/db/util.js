/**
 * Various helper utilities
 */

const r = require('rethinkdb')

/**
 * Shorthand for thread retrieval
 * @param {int} id
 * @returns {Object}
 */
export function getThread(id) {
	return r.table('threads').get(id)
}

/**
 * Remove properties the client should not be seeing
 * @param {Object} post - Post object
 * @returns {Object} - Same post object for convience
 */
export function formatPost(post) {
	for (let key of ['ip', 'deleted', 'imgDeleted', 'nonce']) {
		delete post[key]
	}
    return post
}

/**
 * Count the number of replies in a RethinkDB thread ducument
 * @param {Object} thread
 * @returns {Object}
 */
export function countReplies(thread) {
    return thread('replies').coerceTo('array').count().sub(1)
}
