const admin = require('../server/admin'),
	common = require('../common'),
	util = require('./util')

/**
 * Reads thread and post data from the database
 */
export default class Reader {
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
	 * @param {Object} opts - Extra options
	 * @returns {(Object|null)} - Retrieved post or null
	 */
	async getThread(id, opts = {}) {
		// Verify thread exists
		if (!(await util.parentThread(id)))
			return null
		let thread = this.threadQuery(util.getThread(id))

		// Only show the last N post
		if (opts.abbrev) {
			thread = thread..merge({
				posts: r.row('posts')
					.coerceTo('array')
					.slice(-opts.abbrev + 1)
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
				historyCtr: r.row('history').count(),
				replyCtr: util.countReplies(r.row)
				imageCtr: r.row('posts')
					.coerceTo('array')
					.filter(doc => doc(1).hasFields('image'))
					.count()

				// Ensure we always get the OP
				op: r.row('posts')(r.row('id'))
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
	 * @param {string} orderBy - Index to order the threads by
	 * @returns {Array} - Array of threads
	 */
	async getBoard(orderBy) {
	    const threads = await r.table('threads')
			.getAll(this.board, {index: 'board'})
			.orderBy({index: orderBy})
			.forEach(thread =>
				this.threadQuery(thread)
				.without('posts'))
			.run(rcon)
		for (let i; i < threads.length; i++) {
			const thread = threads[i]
			if (!this.parsePost(thread.op)) {
				threads.splice(i, 1)
				continue
			}
			util.formatPost(thread)
		}
		return threads
	}
}
