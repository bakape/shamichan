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
}
