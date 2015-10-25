/*
 Database comtroller for each connected client
 */

const _ = require('underscore'),
	admin = require('../server/admin'),
	amusement = require('../server/amusement'),
	async = require('async'),
	cache = require('./cache'),
	common = require('../common'),
	config = require('../config'),
	imager = require('../imager'),
	Muggle = require('../util/etc').Muggle,
	r = global.rethink,
	radio = config.RADIO && require('../server/radio'),
	{rcon, redis} = global,
	state = require('../server/state'),
	tripcode = require('bindings')('tripcode')

class ClientController {
	constructor(client) {
		this.client = client
		this.board = client.board
		this.ident = client.ident
	}
	insertPost(msg, cb) {
		if (config.READ_ONLY)
			return cb(Muggle('Can\'t post right now'))
		const {ip} = this.ident,
			{client} = this,
			now = Date.now(),
			{op, image} = msg,
			isThread = !msg.op,
			m = redis.multi()
		const post = {
			ip,
			time: now,
			nonce: msg.nonce
		}
		if (isThread) {
			post.bumpTime = now

			// Stores all updates that happened to the thread, so we can
			// pass them to the client, if they are behind
			post.history = []
		}
		async.waterfall([
			next => {
				if (image) {
					if (!/^\d+$/.test(image))
						return next(Muggle('Expired image token'))
				}

				let body = ''
				const {frag} = msg
				if (frag) {
					if (/^\s*$/g.test(frag))
						return next(Muggle('Bad post body'))
					if (frag.length > common.MAX_POST_CHARS)
						return next(Muggle('Post is too long'))
					body = hot_filter(frag
						.replace(state.hot.EXCLUDE_REGEXP, ''))
				}
				post.body = body

				if (isThread)
					return next(null, null)
				cache.validateOP(op, this.board, next)
			},
			(valid, next) => {
				if (!isThread) {
					if (valid === false)
						return next(Muggle('Thread does not exist'))
					post.op = op
				}
				else {
					if (!image)
						return next(Muggle('Image missing'))
					if (msg.subject) {
						const subject = msg.subject
							.trim()
							.replace(state.hot.EXCLUDE_REGEXP, '')
							.replace(/[「」]/g, '')
							.slice(0, STATE.hot.SUBJECT_MAX_LENGTH)
						if (subject)
							post.subject = subject
					}
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

				if ('auth' in msg) {
					if (!msg.auth
						|| !client.ident
						|| msg.auth !== client.ident.auth
					)
						return next(Muggle('Bad auth'))
					post.auth = msg.auth
				}

				if (isThread)
					this.checkThrottle(next)
				else
					this.checkThreadLocked(op, next)
			},
			next =>
				r.table('main').get('info')
					.update({post_ctr: r.row('post_ctr').default(0).add(1)},
						{returnChanges: true})
					('changes')('new_val')('post_ctr')(0)
					.run(rcon, next),
			(id, next) => {
				if (!client.synced)
					return next(Muggle('Dropped; post aborted'))
				if (client.post)
					return next(Muggle('Already have a post'))

				amusement.roll_dice(body, post)
				client.post = post
				post.id = id
				const links = extractLinks(post.body)
				this.validateLinks(links, next)
			},
			(links, next) => {
				if (links)
					post.links = links
				imager.obtain_image_alloc(image, next)
			},
			(image, next) => {
				if (image) {
					post.image = image.image
					if (isThread && post.image.pinky)
						return next(Muggle('Image is the wrong size'))
					delete post.image.pinky;
					this.imageDuplicateHash(m, msg.image.hash, id)
					return imager.commit_image_alloc(image, next)
				}
				next()
			},
			next =>
				this['write' + (isThread ? 'Thread' : 'reply')](post, next),
			(res, next) => {
				// Set of currently open posts
				m.sadd('liveposts', id)

				// Threads have their own post number as the OP
				const channel = op || id
				cache.cache(m, id, channel, this.board)

				// For priveledged authenticated clients only
				let augments
				const mnemonic = admin.genMnemonic(ip)
				if (mnemonic)
					augments = {mod: mnemonic}
				formatPost(post)
				this.publish(m, channel, [common.INSERT_POST, post], augments)
				m.exec(next)
			}
		], err => {
			if (err && client.post === post)
				client.post = null
			cb(err)
		})
	}
	checkThreadLocked(op, cb) {
		r.table('threads').get(op)('locked').default(false)
			.run(rcon, (err, lock) =>
				cb(err || lock && Muggle('Thread is locked')))
	}
	// Check if IP has not created a thread recently to prevent spam
	checkThrottle(cb) {
		// So we can spam new threads in debug mode
		if (config.DEBUG)
			return cb()
		redis.exists(`ip:${this.ident.ip}:throttle`, (err, exists) =>
			cb(err || exists && Muggle('Too soon')))
	}
	// Parse post fragment and return an array of posts linked
	extractLinks(frag) {
		const links = []
		const onee = new common.OneeSama({
			tamashii(num) {
				links.push(num)
			}
		})
		// TEMP: Dummy model
		onee.setModel({}).fragment(frag)
		return links
	}
	validateLinks(nums, cb) {
		if (!nums.length)
			return cb(null, null)
		async.waterfall([
			next => {
				const m = redis.multi()
				for (let num of nums) {
					m.hget(`threads:num`)
				}
				m.exec(next)
			},
			(threads, next) => {
				const links = {}
				for (let i = 0; i < threads.length; i++) {
					if (threads[i])
						links[nums[i]] = threads[i]
				}
				next(null, _.isEmpty(links) ? null : links)
			}
		], cb)
	}
	imageDuplicateHash(m, hash, num) {
		m.zadd('imageDups', Date.now() + (config.DEBUG ? 30000 : 3600000),
			`${num}:${hash}`)
	}
	writeThread(post, cb) {
		// Prevent thread spam
		m.setex(`ip:${ip}:throttle`, config.THREAD_THROTTLE, op)
		r.table('threads').insert(post).run(rcon, cb)
	}
	writeReply(post, cb) {
		async.waterfall([
			next =>
				r.table('replies').insert(post).run(rcon, next),
			// Bump the thread up to the top of the board
			(res, next) => {
				if (common.is_sage(post.email))
					return next()
				r.branch(
					// Verify not over bump limit
					r.table('replies')
						.getAll(post.op, {index: 'op'})
						.count()
						.lt(config.BUMP_LIMIT[this.board]),
					r.table('threads').get(post.op)
						.update({bumpTime: Date.now()}),
					null
				).run(rcon, next)
			}
		], cb)
	}
	publish(m, chan, msg, augments) {
		msg = [msg]
		if (augments)
			msg.push(augments)
		m.publish(chan, JSON.stringify(msg))
	}
}

// Regex replacement filter
function hot_filter(frag) {
	let filter = state.hot.FILTER
	if (!filter)
		return frag
	for (let f of filter) {
		const m = frag.match(f.p)
		if (m) {
			// Case sensitivity
			if (m[0].length > 2) {
				if (/[A-Z]/.test(m[0].charAt(1)))
					f.r = f.r.toUpperCase()
				else if (/[A-Z]/.test(m[0].charAt(0)))
					f.r = f.r.charAt(0).toUpperCase() + f.r.slice(1)
			}
			return frag.replace(f.p, f.r)
		}
	}
	return frag
}

// Remove properties the client should not be seeing
function formatPost(post) {
	// Only used internally and should not be exposed to clients
	for (let key of ['ip', 'deleted', 'imgDeleted']) {
		delete post[key];
	}
}
