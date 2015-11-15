const admin = require('../server/admin'),
	amusement = require('../server/amusement'),
	cache = require('./cache'),
	common = require('../common'),
	config = require('../config'),
	imager = require('../imager'),
	Muggle = require('../util/etc').Muggle,
	r = require('rethinkdb'),
	radio = config.RADIO && require('../server/radio'),
	{rcon, redis} = global,
	state = require('../server/state'),
	tripcode = require('bindings')('tripcode')

/** Performs approprite database I/O in response to client websocket messages */
class ClientController {
	/**
	 * Create a database controller
	 * @param {Client} client
	 */
	constructor(client) {
		this.client = client
		this.board = client.board
		this.ident = client.ident
	}

	/**
	 * Insert post into the database
	 * @param {Array} msg
	 */
	async insertPost(msg) {
		if (config.READ_ONLY)
			throw Muggle('Can\'t post right now')
		const {client} = this,
			{op, image} = msg,
			isThread = !msg.op
		const post = {
			ip: this.ident.ip,
			time: Date.now(),
			nonce: msg.nonce,
			board: this.board
		}
		if (image && !/^\d+$/.test(image))
			throw Muggle('Expired image token')

		if (isThread)
			await this.prepareThread(msg, post)
		else
			await this.prepareReply(op, post)


		if (!client.synced)
			throw Muggle('Dropped; post aborted')
		if (client.post)
			throw Muggle('Already have a post')
		post.id = await r.table('main').get('info')
			.update({post_ctr: r.row('post_ctr').add(1)},
				{returnChanges: true})
			('changes')('new_val')('post_ctr')(0)
			.run(rcon)
		client.post = post
		amusement.roll_dice(body, post)
		const [parsedBody, links] = await this.parsePost(msg)
		post.body = parsedBody
		post.length = postLength(parsedBody)

		const m = redis.multi()
		if (image) {
			const alloc = await obtainImageAlloc(image)
			image = post.image = alloc.image
			if (isThread && image.pinky)
				throw Muggle('Image is the wrong size')
			delete image.pinky;
			this.imageDuplicateHash(m, image.hash, id)
			await commitImageAlloc(alloc)
		}

		if (isThread)
			await this.writeThread(m)
		else
			await this.writeReply()
		await this.puglishPost(m)
		await this.backlinks(links)
	}

	/**
	 * Perform reply-specific validations
	 * @param {int} op
	 * @param {Object} post
	 */
	async prepareReply(op, post) {
		if (await cache.validateOP(op, this.board) === false)
			throw Muggle('Thread does not exist')
		await this.checkThrottle()
		post.op = op
	}

	/**
	 * Perform thread-specific validations and parsing
	 * @param {Object} msg
	 * @param {Object} post
	 */
	async prepareThread(msg, post) {
		if (!msg.image)
			throw Muggle('Image missing')
		await this.checkThreadLocked(op)
		if (msg.subject) {
			const subject = msg.subject
				.trim()
				.replace(state.hot.EXCLUDE_REGEXP, '')
				.replace(/[「」]/g, '')
				.slice(0, STATE.hot.SUBJECT_MAX_LENGTH)
			if (subject)
				post.subject = subject
		}
		post.bumpTime = Date.now()

		// Stores all updates that happened to the thread, so we can
		// pass them to the client, if they are behind
		post.history = []
	}

	/**
	 * Ensure a thread is not locked
	 * @param {int} op
	 */
	async checkThreadLocked(op) {
		if (await getPost(op)('locked').default(false).run(rcon))
			throw Muggle('Thread is locked')
	}

	/**
	 * Check if IP has not created a thread recently to prevent spam
	 */
	async checkThrottle() {
		// So we can spam new threads in debug mode
		if (config.DEBUG)
			return
		if (await redis.existsAsync(`ip:${this.ident.ip}:throttle`))
			throw Muggle('Too soon')
	}

	/**
	 * Parse message contents into post object and validate
	 * @param msg
	 * @returns {[Array,Object]}
	 */
	async parsePost(msg) {
		const {post} = this.client
		if ('auth' in msg) {
			if (!msg.auth
				|| !client.ident
				|| msg.auth !== client.ident.auth
			)
				throw Muggle('Bad auth')
			post.auth = msg.auth
		}

		let body = ''
		const {frag} = msg
		if (frag) {
			if (/^\s*$/g.test(frag))
				throw Muggle('Bad post body')
			if (frag.length > common.MAX_POST_CHARS)
				throw Muggle('Post is too long')
			body = amusement.hot_filter(frag
				.replace(state.hot.EXCLUDE_REGEXP, ''))
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

		return await this.parseFragment(body)
	}

	/**
	 * Parse text body message fragment string
	 * @param {string} frag
	 * @returns {[Array,Object]}
	 */
	async parseFragment(frag) {
		const m = frag.match(/>>\d+/g)
		frag = frag.split(' ')
		if (!m)
			return [frag, {}]

		// Validate links and determine their parent board and thread
		const links = [],
			m = redis.multi()
		m.forEach(link => links.push(link.slice(2)))
		links.forEach(num => cache.getParenthood(m, num))
		const res = await m.execAsync(),
			confirmed = {}
		for (let i = 0; i < res.length; i += 2) {
			const board = res[i],
				thread = res[i + 1]
			if (board && thread)
				confirmed[links[i / 2]] = [board, parseInt(thread)]
		}

		// Insert post links and hash commonds as tumples into the text body
		// array
		const parsed = []
		frag.forEach(word => this.injectLink(word, parsed, confirmed)
			|| amusement.roll_dice(word, parsed)
			|| parsed.push(word))
		return [parsed, confirmed]
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
	 * Write the hash of an image to databse to later check for duplicates
	 * against
	 * @param {redis.multi} m
	 * @param {string} hash
	 * @param {int} num
	 */
	imageDuplicateHash(m, hash, num) {
		m.zadd('imageDups', Date.now() + (config.DEBUG ? 30000 : 3600000),
			`${num}:${hash}`)
	}

	/**
	 * Write thread to database
	 * @param {redis.multi} m
	 */
	async writeThread(m) {
		// Prevent thread spam
		m.setex(`ip:${ip}:throttle`, config.THREAD_THROTTLE, op)
		await this.writePost()
	}

	/**
	 * Write post to database
	 */
	async writePost() {
		const {post} = this.client

		// Useless client-side
		if (post.image)
			delete post.image.hash
		await r.table('posts').insert(post).run(rcon)
	}

	/**
	 * Write reply to database and bump parent thread, if needed
	 */
	async writeReply() {
		const {post} = this.client
		await this.writePost()

		// Bump the thread up to the top of the board
		if (common.is_sage(post.email))
			return
		await r.branch(
			// Verify not over bump limit
			r.table('posts')
				.getAll(post.op, {index: 'op'})
				.count()
				.lt(config.BUMP_LIMIT[this.board]),
			getPost(post.op).update({bumpTime: Date.now()}),
			null
		).run(rcon)
	}

	/**
	 * Publish newly created post to live clients
	 * @param {redis.multi} m
	 */
	async puglishPost(m) {
		// Set of currently open posts
		m.sadd('liveposts', id)

		// Threads have their own id as the op property
		const {post} = this.client,
			channel = threadNumber(post)
		cache.cache(m, post.id, channel, this.board)
		const msg = [[common.INSERT_POST, post]]

		// For priveledged authenticated clients only
		const mnemonic = admin.genMnemonic(post.ip)
		if (mnemonic)
			msg.push({mod: mnemonic})
		formatPost(post)
		await this.publish(m, channel, msg)
	}

	/**
	 *  Store message inside the replication log and publish to connected
	 *  clients through redis
	 * @param {redis.multi} m
	 * @param {int} op
	 * @param {[[]]} msg
	 */
	async publish(m, op, msg) {
		// Ensure thread exists, because the client in some cases publishes
		// to external threads
		if (await postsExists(op))
			return
		msg = JSON.stringify(msg)
		await getPost(op).update({
			history: r.row('history').append(msg)
		}).run(rcon)
		m.publish(op, msg)
		await m.execAsync()
	}

	/**
	 * Write this post's location data to the post we are linking
	 * @param {Object} links
	 */
	async backlinks(links) {
		const {post} = this.client
		for (let num in links) {
			const [board, op] = links[num]

			// Coerce to integer
			num = +num
			const update = {
				backlinks: {
					[post.id]: [this.board, post.op]
				}
			}

			// Ensure target post exists
			if (await getPost(num).eq(null).run(rcon))
				continue
			await getPost(num).update(update).run(rcon)
			await this.publish(redis.multi(), op,
				[[common.UPDATE_POST, update]])
		}
	}

	/**
	 * Append to the text body of a post
	 * @param {string} frag
	 * @returns {Promise}
	 */
	async appendPost(frag) {
		const [body, links] = await this.parseFragment(frag),
			{post} = this.client
		await getPost(post.id).update({
			body: r.row('body').concat(body)
		}).run(rcon)

		// Persist to memory as well
		post.body = post.body.concat(body)
		post.length += postLength(body)
		await this.publish(redis.multi(), threadNumber(post),
			[[common.UPDATE_POST, post.id, {body}]])
		await this.backlinks(links)
	}

	/**
	 * Insert image into an existing post
	 * @param {string} id
	 */
	async insertImage(id) {
		const alloc = await obtainImageAlloc(id),
			{post} = client
		if (!post.op)
			throw Muggle('Can\'t add another image to an OP')
		const {image} = alloc
		if (!image.pinky)
			throw Muggle('Image is wrong size')
		delete image.pinky
		if (!(await postsExists(post.id)))
			throw Muggle('Post does not exist')

		await commitImageAlloc(alloc)
		const m = redis.multi()
		this.imageDuplicateHash(m, image.hash, post.id)
		await getPost(post.id).update({image}).run(rcon)

		// Useless client-side
		delete image.hash
		await this.publish(m, post.op, [[common.UPDATE_POST, {image}]])
	}
}

/**
 * Remove properties the client should not be seeing
 * @param {Object} post
 */
function formatPost(post) {
	for (let key of ['ip', 'deleted', 'imgDeleted']) {
		delete post[key];
	}
}

/**
 * Shorthand for post retrieval
 * @param num
 * @returns {*}
 */
function getPost(num) {
	return r.table('posts').get(num)
}

/**
 * Check if post exists in the database
 * @param {int} id
 */
async function postsExists(id) {
    await getPost(num).eq(null).not().run(rcon)
}

/**
 * Detect a posts parent thread. Needed because OP's do not have an 'op'
 * property.
 * @param {Object} post
 * @returns {int}
 */
function threadNumber(post) {
	return post.op || post.id
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

/**
 * Read image allocation data with supplied id from database
 * @param {string} id
 * @returns {Object}
 */
async function obtainImageAlloc(id) {
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
		const fnm = alloc.tmps[dir]
		if (!/^[\w_]+$/.test(fnm))
			throw Muggle('Suspicious filename: ' + JSON.stringify(fnm))
	}
	return alloc
}

/**
 * Copy image files from temporary folders to permanent served ones
 * @param {Object} alloc
 */
async function commitImageAlloc(alloc) {
	for (let kind in alloc.tmps) {
		await etc.copyAsync(imager.media_path('tmp', alloc.tmps[kind]),
			imager.media_path(kind, alloc.image[kind]))
	}

	// We should already hold the lock at this point.
	const key = 'image:' + alloc.id,
		m = redis.multi()
	m.del(key)
	m.del('lock:' + key)
	await m.execAsync()
}
