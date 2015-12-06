const common = require('../common'),
	index = require('./index'),
	{EventEmitter} = require('events'),
	winston = require('winston')

/**
 * Listens to redis channels and parses and dispatches messages to client by
 * access right
 */
class Subscription extends EventEmitter {
	/**
	 * Construct new Redis listener/parser
	 * @param {int} key
	 */
	constructor(key) {
		super()
		this.setMaxListeners(0)

		// Redis will swicth into dedicated subscription mode on
		// .subscribe(), so we need a separate client for each subscription
		const redis = index.redisClient()
		redis.on('error', err => this.onError(err))
		redis.on('message', msg => this.onMessage(msg))
		redis.subscribe(key)
		this.redis = redis
		this.key = key
	}

	/**
	 * Log error and kill listener
	 * @param {Error} err
	 */
	onError(err) {
		winston.error('Subscription error: ', err)
		this.commitSudoku()
	}

	/**
	 * Remove all references to allow instance garbage collection
	 */
	commitSudoku() {
		this.removeAllListeners().redis.unsubscribe()
		delete Subscription.keys[this.key]
	}

	/**
	 * Subscribe to an existing Subscription() object for this channel or
	 * create a new one, if none
	 * @param {int} thread
	 * @param {Client} client
	 */
	static get(thread, client) {
		(Subscription.keys[thread] || new Subscription(thread)).listen(client)
	}

	/**
	 * Listen to messages on the appriate priveledge channel
	 * @param {Client} client
	 */
	listen(client) {
		let priv = 'normal'
		if (common.checkAuth('moderator', client.ident))
			priv = 'mod'
		else if (common.checkAuth('janitor', client.ident))
			priv = 'janny'

		const handler = msg => client.send(msg)
		this.on(priv, handler)
		client.once('close', () =>
			this.removeListener(priv, handler).checkCount())
	}

	/**
	 * Kill Subscription() after delay, if there are no listeners
	 */
	checkCount() {
		if (this.countListeners())
			return
		if (this.idleOutTimer)
			clearTimeout(this.idleOutTimer)
		this.idleOutTimer = setTimeout(() =>
			!this.countListeners() && this.commitSudoku(), 30000)
	}

	/**
	 * Summ the listeners counts on all subchannels
	 * @returns {int}
	 */
	countListeners() {
		let count = 0
		for (let priv of ['normal', 'janny', 'mod']) {
			count += this.listenerCount(priv)
		}
		return count
	}

	/**
	 * Parse Redis subscription message and emit appropriate message to all
	 * access levels
	 * @param {string} unparsed
	 */
	onMessage(unparsed) {
		const [msg, extra] = JSON.parse(unparsed)
		this.emit('normal', msg)

		// Bundle additional info for priveledged clients
		for (let priv of ['janny', 'mod']) {
			const extended = extra && extra[priv] && msg.concat([extra[priv]])
			this.emit(priv, extended || msg)
		}
	}
}
module.exports = Subscription

// Can't define static properties with ES6 classes. ES7 when?
Subscription.keys = {}
