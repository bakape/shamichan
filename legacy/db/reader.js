const admin = require('../server/admin'),
	common = require('../common'),
	r = require('rethinkdb'),
	{rcon} = global,
	util = require('./util')

/**
 * Reads thread and post data from the database
 */
class Reader {
	/**
	 * Constructs new database reader
	 * @param {string} board
	 * @param {Object} ident
	 */
	constructor(board, ident) {
		this.ident = ident
		this.board = board
		if (common.checkAuth('janitor', ident)) {
			this.hasAuth = true
			this.canModerate = common.checkAuth('moderator', ident)
		}
	}

	/**
	 * Retrieve thread JSON from the database
	 * @param {int} id - Thread id
	 * @param {int} opts - Last N posts to fetch. 0 fetches all.
	 * @returns {(Object|null)} - Retrieved post or null
	 */
	async getThread(id, lastN) {
		// Verify thread exists
		if (!(await util.parentThread(id)))
			return null
		let thread = this.threadQuery(util.getThread(id))

		// Only show the last N post
		if (lastN) {
			thread = thread.merge({
				posts: thread('posts')
					.coerceTo('array')
					.slice(-lastN + 1)
					.coerceTo('object')
			})
		}
		thread = await thread.run(rcon)

		// Verify thread access rights
		if (!this.parsePost(thread.op))
			return null
		util.formatPost(thread)

		// Delete duplicate OP post object, if any
		delete thread.posts[thread.id]
		for (let id in thread.posts) {
			if (!this.parsePost(thread[id]))
				delete thread[id]
		}
		return thread
	}

	/**
	 * Common part of a all thread queries
	 * @param {Object} thread
	 * @returns {Object}
	 */
	threadQuery(thread) {
		return thread.merge({
				historyCtr: thread('history').count(),
				replyCtr: util.countReplies(thread),
				imageCtr: thread('posts')
					.coerceTo('array')
					.filter(doc => doc(1).hasFields('image'))
					.count(),

				// Ensure we always get the OP
				op: thread('posts')(thread('id'))
			})
			.without('history')
	}

	/**
	 * Read a single post from the database
	 * @param {int} id - Post id
	 * @returns {(Object|null)} - Retrieved post or null
	 */
	async getPost(id) {
		const op = await util.parentThread(id)
		if (!op)
			return null
		return this.parsePost(await util.getThread(op)
			('threads')(id)
			.run(rcon))
	}

	/**
	 * Adjust post according to the reading client's access priveledges
	 * @param {Object} post - Post object
	 * @returns {(Object|null)} - Parsed post object or null, if client not
	 * 	allowed to view post
	 */
	parsePost(post) {
		if (!post)
			return null
		if (!this.hasAuth) {
			if (post.deleted)
				return null
			if (post.imgDeleted)
				delete post.image
			delete post.mod
		}
		if (this.canModerate) {
			const mnemonic = admin.genMnemonic(post.ip)
			if (mnemonic)
				post.mnemnic = mnemonic
		}
		return util.formatPost(post)
	}

	/**
	 * Retrieve all threads on the board with their OPs
	 * @returns {Array} - Array of threads
	 */
	async getBoard() {
		// Current limitation of Babel.js in async functions
		let self = this
	    const threads = await r.table('threads')
			.getAll(this.board, {index: 'board'})
			.forEach(thread =>
				self.threadQuery(thread)
				.without('posts'))
			.run(rcon)
		for (let num in threads) {
		    const thread = threads[num]
			if (!this.parsePost(thread.op)) {
				delete threads[num]
				continue
			}
			util.formatPost(thread)
		}
		return threads
	}
}
module.exports = Reader
