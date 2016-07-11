/**
 * Performs approprite database I/O in response to client websocket messages
 */
class ClientController {
	/**
	 * Create a client database controller
	 * @param {Client} client
	 */
	constructor(client) {
		this.client = client
		this.board = client.board
		this.ident = client.ident
		this.op = client.op
	}

	/**
	 * Ensure client did not disconnect midway
	 */
	checkSynced() {
		if (!this.client.synced)
			throw Muggle('Dropped; post aborted')
	}

	/**
	 * Insert post into the database
	 * @param {Array} msg
	 */
	async insertPost(msg) {
		const {op} = this.client
		const post = {
			ip: this.ident.ip,
			op,
			time: Date.now(),
			nonce: msg.nonce,
			editing: true
		}
		if (!(await cache.validateOP(op, this.board)))
			throw Muggle('Thread does not exist')
		await this.checkThreadLocked()
		await this.assignPostID(post)
		this.parseName(msg)
		const links = await this.parseBody(msg),
			m = redis.multi()
		post.image = await this.allocateImage(msg.image, m, false)

		// Message for live publishing and storage in replication log
		let publishMsg = [[common.INSERT_POST, util.formatPost(_.clone(post))]]

		// For priveledged authenticated clients only
		const mnemonic = admin.genMnemonic(post.ip)
		if (mnemonic)
			msg.push({mod: mnemonic})
		publishMsg = JSON.stringify(publishMsg)

		// Write to database
		await Promise.join(
			util.getThread(this.op).update({
				history: r.row('history').append(publishMsg),
				replies: {
					[post.id]: post
				}
			}).run(rcon),
			this.boardCounter(this.board)
		)

		cache.add(m, post.id, this.op, this.board)
		await Promise.join(
			this.publish(m, this.op, publishMsg),
			this.bumpThread(),
			this.backlinks(links)
		)
	}

	/**
	 * Ensure thread is not locked
	 */
	async checkThreadLocked() {
		if (await util.getThread(this.op)
			('locked')
			.default(false)
			.run(rcon)
		)
			throw Muggle('Thread is locked')
	}

	/**
	 * Bump the thread up to the top of the board, if needed
	 */
	async bumpThread() {
		if (common.is_sage(this.client.post.email))
			return
		await uti.getThread(this.op).do(thread =>
			r.branch(
				// Verify not over bump limit
				util.countReplies(thread).lt(config.BUMP_LIMIT[this.board]),
				thread.update({bumpTime: Date.now()}),
				null
			)
		).run(rcon)
	}

	/**
	 * Publish message to connected clients through redis and perform all other
	 * queued redis operations
	 * @param {redis.multi} m1
	 * @param {int} op
	 * @param {string} msg
	 */
	async publish(m, op, msg) {
		m.publish(op, msg)
		await m.execAsync()
	}

	/**
	 * Update post hash in rethinkDB, append to replication log and publish to
	 * live clients
	 * @param {int} id - Post ID
	 * @param {int} op - Post thread
	 * @param {Object} update - Update toinject into post
	 * @param {sting} [live=updateMessage(id,update)] - Optional separate update
	 * for live publishes in cases when the update includes rethinkDB
	 * expressions
	 */
	async updatePost(id, op, update, live = updateMessage(id, update)) {
		live = JSON.stringify(live)
		await Promise.join(
			util.getThread(op).update({
				history: r.row('history').append(live),
				posts: {
					[id]: update
				}
			}).run(rcon),
			this.boardCounter(this.board)
		)
		await redis.publishAsync(op, live)
	}

	/**
	 * Append to the text body of a post
	 * @param {string} frag
	 */
	async appendPost(frag) {
		const [body, links] = await this.parseFragment(frag),
			{post} = this.client
		await this.updatePost(post.id, this.op,
			{body: r.row('body').concat(body)},
			{body})

		// Persist to memory as well
		post.body = post.body.concat(body)
		post.length += postLength(body)
		await this.backlinks(links)
	}

	/**
	 * Finish the current open post
	 */
	async finishPost() {
		const {id} = this.client.post
		delete this.client.post
		await Promise.join(this.updatePost(id, this.op, {editing: false}),
			// Remove from open post set
			redis.sremAsync('liveposts', id))
	}
}
module.exports = ClientController

/**
 * Shorthand for creating an update message
 * @param {int} id
 * @param {Object} update
 * @returns {[[]]}
 */
 function updateMessage(id, update) {
    return [[common.UPDATE_POST, id, update]]
 }

/**
 * Get length of post text body array + strings inside it
 * @param {Array} body
 */
function postLength(body) {
	let length = body.length
	for (let frag of body) {
		if (typeof frag !== 'string')
			continue

		// String length can be zero due to filters
		const wordLength = frag.length - 1
		if (frag.length > 0)
			length += wordLength
	}
}
