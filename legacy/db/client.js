const _ = require('underscore'),
	admin = require('../server/admin'),
	amusement = require('../server/amusement'),
	cache = require('./cache'),
	common = require('../common'),
	config = require('../config'),
	imager = require('../imager'),
	Muggle = require('../util/etc').Muggle,
	Promise = require('bluebird'),
	r = require('rethinkdb'),
	radio = config.RADIO && require('../server/radio'),
	{rcon, redis} = global,
	state = require('../server/state'),
	tripcode = require('bindings')('tripcode'),
	util = require('./util')

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
	 * Insert thread into DB
	 * @param {Object} msg
	 */
	async insertThread(msg) {
		// Check if IP has not created a thread recently to prevent spam
		if (!config.DEBUG
			&& await redis.existsAsync(`ip:${this.ident.ip}:throttle`)
		)
			throw Muggle('Too soon')
		const {client} = this,
			now = Date.now(),
			{ip} = this.ident

		// A thread is just a container for posts with metadata attached. The OP
		// is actually stored as a post in the thread with the same ID as the
		// thread.
		const thread = {
			ip,
			time: now,
			nonce: msg.nonce,
			bumpTime: now,
			board: this.board,
			posts: {},

			// Stores all updates that happened to the thread, so we can
			// pass them to the client, if they are behind
			history: []
		}
		const post = {
			ip,
			time: now,
			nonce: msg.nonce,
			editing: true
		}
		const id = await this.assignPostID(post)
		thread.id = post.op = id
		thread.posts[id] = post

		this.parseName(msg)
		if (msg.subject) {
			const subject = msg.subject
				.trim()
				.replace(state.hot.EXCLUDE_REGEXP, '')
				.replace(/[「」]/g, '')
				.slice(0, STATE.hot.SUBJECT_MAX_LENGTH)
			if (subject)
				post.subject = subject
		}
		client.postLength = 0
		const m = redis.multi()
		post.image = await this.allocateImage(msg.image, m, true)
		await Promise.join(r.table('threads').insert(thread).run(rcon),
			this.boardCounter(this.board))

		// Prevent thread spam
		m.setex(`ip:${ip}:throttle`, config.THREAD_THROTTLE, post.id)
		cache.add(m, post.id, post.id, this.board)
		await m.execAsync()

		// Redirect the client to the new thread
		client.send([common.REDIRECT, this.board, post.id, true])
	}

	/**
	 * Incerement post counter in the DB and assign to the new post
	 * @param {Object} post
	 * @returns {int} - Post ID
	 */
	async assignPostID(post) {
		this.checkSynced()
		this.client.post = post
		return post.id = await r.table('main').get('info')
			.update({post_ctr: r.row('post_ctr').add(1)},
				{returnChanges: true})
			('changes')('new_val')('post_ctr')(0)
			.run(rcon)
	}

	/**
	 * Ensure client did not disconnect midway
	 */
	checkSynced() {
		if (!this.client.synced)
			throw Muggle('Dropped; post aborted')
	}

	/**
	 * Parse post name, tipcode, email and titles and assign to post object
	 * @param {Object} msg
	 */
	parseName(msg) {
		const {post} = this.client
		if ('auth' in msg) {
			if (!msg.auth
				|| !client.ident
				|| msg.auth !== client.ident.auth
			)
				throw Muggle('Bad auth')
			post.auth = msg.auth
		}

		// Replace names, when a song plays on r/a/dio
		if (radio && radio.name)
			post.name = radio.name
		else if (!state.hot.forced_anon) {
			if (msg.name) {
				const parsed = common.parse_name(msg.name)
				post.name = parsed[0]
				const spec = state.hot.SPECIAL_TRIPCODES
				if (spec && parsed[1] && parsed[1] in spec)
					post.trip = spec[parsed[1]]
				else if (parsed[1] || parsed[2]) {
					const trip = tripcode.hash(parsed[1], parsed[2])
					if (trip)
						post.trip = trip
				}
			}
			if (msg.email)
				post.email = msg.email.trim().substr(0, 320)
		}
	}

	/**
	 * Allocate a processed image and its thumbnails to be served with a post
	 * @param {int} id
	 * @param {redis.multi} m
	 * @param {boolean} isThread
	 * @returns {Object} - Image object
	 */
	async allocateImage(id, m, isThread) {
		const alloc = await obtainImageAlloc(id),
			{image} = alloc
		if (isThread && image.pinky)
			throw Muggle('Image is the wrong size')
		delete image.pinky

		/*
		 Write the perceptual hash of an image to a specialised sorted set to
		 later check for duplicates against
		*/
		const till = Date.now() + (config.DEBUG ? 30000 : 3600000)
		m.zadd('imageDups', till, `${num}:${image.hash}`)

		// Useless after image hash has been written
		delete image.hash
		await commitImageAlloc(alloc)
		return image
	}

	/**
	 * Read image allocation data with supplied id from database
	 * @param {string} id
	 * @returns {Object}
	 */
	async obtainImageAlloc(id) {
		const m = redis.multi(),
			key = 'image:' + id
		m.get(key)
		m.setnx('lock:' + key, '1');
		m.expire('lock:' + key, 60);
		let [alloc, status] = await m.execAsync()
		if (status !== '1')
			throw Muggle('Image in use')
		if (!alloc)
			throw Muggle('Image lost')
		alloc = JSON.parse(res[0])
		alloc.id = id

		// Validate allocation request
		if (!alloc || !alloc.image || !alloc.tmps)
			throw Muggle('Invalid image alloc')
		for (let dir in alloc.tmps) {
			const fileName = alloc.tmps[dir]
			if (!/^[\w_]+$/.test(fileName))
				throw Muggle(`Suspicious filename: ${JSON.stringify(fileName)}`
		}
		return alloc
	}

	/**
	 * Copy image files from temporary folders to permanent served ones
	 * @param {Object} alloc
	 */
	async commitImageAlloc(alloc) {
		const tasks = []
		for (let kind in alloc.tmps) {
			tasks.push(etc.copyAsync(imager.media_path('tmp', alloc.tmps[kind]),
				imager.media_path(kind, alloc.image[kind])))
		}
		await Promise.all(tasks).catch(err =>
			throw Muggle('Couldn\'t copy file into place:', err))

		// We should already hold the lock at this point.
		const key = 'image:' + alloc.id,
			m = redis.multi()
		m.del(key)
		m.del('lock:' + key)
		await m.execAsync()
	}

	/**
	 * Increment the history counter of the board, which is used to generate
	 * e-tags
	 * @param {string} board
	 */
	async boardCounter(board) {
	    await r.('main').get('boardCtrs').update({
			[board]: r.row(board).default(0).add(1)
		}).run(rcon)
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
	 * Parse message text body into post object
	 * @param {Object} msg
	 * @returns {Object} - Confirmed post links inside text
	 */
	async parseBody(msg) {
		const {post} = this.client,
			{frag} = msg
		let body = ''
		if (frag) {
			if (/^\s*$/g.test(frag))
				throw Muggle('Bad post body')
			if (frag.length > common.MAX_POST_CHARS)
				throw Muggle('Post is too long')
			body = amusement.hot_filter(frag
				.replace(state.hot.EXCLUDE_REGEXP, ''))
		}

		const [parsed, links] = await this.parseFragment(body)
		post.body = parsed
		this.client.postLength = postLength(parsed)
		return links
	}

	/**
	 * Parse text body message fragment string
	 * @param {string} frag
	 * @returns {[Array,Object]} - Parsed post body array and confirmed post
	 * 	links it contains
	 */
	async parseFragment(frag) {
		const m = frag.match(/>>\d+/g)
		frag = frag.split(' ')
		if (!m)
			return [frag, {}]

		// Validate links and determine their parent board and thread
		const links = [],
			multi = redis.multi()
		m.forEach(link => links.push(link.slice(2)))
		links.forEach(num => cache.getParenthood(multi, num))
		const res = await multi.execAsync(),
			confirmed = {}
		for (let i = 0; i < res.length; i += 2) {
			const board = res[i],
				thread = res[i + 1]
			if (board && thread)
				confirmed[links[i / 2]] = [board, parseInt(thread)]
		}

		// Insert post links and hash commands as tumples into the text body
		// array
		const parsed = []
		frag.forEach(word => this.injectLink(word, parsed, confirmed)
			|| amusement.roll_dice(word, parsed)
			|| parsed.push(word))
		return [parsed, confimed]
	}

	/**
	 * Insert links to other posts as tuples into the text body array
	 * @param {string} word - Word to parse
	 * @param {Array} parsed - Array to fill with parse results
	 * @param {Object} confirmed - Object of confirmed links to posts
	 * @returns {boolean} - Link matched
	 */
	injectLink(word, parsed, confirmed) {
		const m = word.match(/^(>{2,})(\d+)$/)
		if (!m)
			return false
		const link = confirmed[m[2]]
		if (!link)
			return false

		// Separate leadind />+/ for qoutes
		if (m[1].length > 2)
			parsed.push(m[1].slice(2))
		parsed.push([common.tupleTypes.link, ...link])
		return true
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
	 * Write this post's location data to the post we are linking
	 * @param {Object} links
	 */
	async backlinks(links) {
		const {id} = this.client.post

		// Run all operations in parallel
		await Promise.all(Object.keys(links).map(async num => {
			const [board, op] = links[num]

			// Coerce to integer
			num = +num
			const op = await cache.parentThread(num)

			// Fail silently, because it does not effect the source post
			if (!op)
				continue
			await Promise.join(
				this.updatePost(num, op, {
					backlinks: {
						[id]: [this.board, this.op]
					}
				}),
				this.boardCounter(board)
			)
		}))
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
	 * Insert image into an existing post
	 * @param {string} id
	 */
	async insertImage(id) {
		const {post} = this.client,
			m = redis.multi(),
			image = post.image
				= await this.allocateImage(id, redis.multi(), false)
		await m.execAsync()
		await this.updatePost(post.id, this.op, {image})
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
