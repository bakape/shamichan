const _ = require('underscore'),
	admin = require('../server/admin'),
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
		if (!(await util.parentThread(id)))
			return null
		const thread = await util.getThread(id)
			.merge({
				historyCtr: r.row('history').count(),
				replyCtr: util.countReplies(r.row)
				imageCtr: r.row('posts')
					.coerceTo('array')
					.filter(doc => doc(1).hasFields('image'))
					.count()
			})
			.without('history')
			.run(rcon)

		// Verify thread OP access rights
		if (!formatPost(_.clone(thread.posts[thread.id])))
			return null
		util.formatPost(thread)
		for (let id in thread.posts) {
			if (this.parsePost(thread[id]))
				delete thread[id]
		}
		return thread
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
		return this.parsePost(await util.getThread(op)('posts')('id').run(rcon))
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
}
