/*
Subsctiption handler
 */

const common = require('../common'),
	index = require('./index'),
	{EventEmitter} = require('events'),
	winston = require('winston');

class Subscription extends EventEmitter {
	constructor(key) {
		super();
		this.setMaxListeners(0);
		const redis = index.redisClient();
		redis.on('error', err => this.onError(err));
		redis.on('message', msg => this.onMessage(msg));
		redis.subscribe(key);
		this.redis = redis;
		this.key = key;
	}
	onError(err) {
		winston.error('Subscription error: ', err);
		this.commitSudoku();
	}
	// Remove all references to allow instance garbage collection
	commitSudoku() {
		this.removeAllListeners().redis.unsubscribe();
		delete Subscription.keys[this.key];
	}
	static get(board, thread, client) {
		// If an instance listening to this redis channel already exists, we
		// can just use that, instead of creating a new one.
		const key = `${board}:${thread}`;
		(Subscription.keys[key] || new Subscription(key)).listen(client);
	}
	listen(client) {
		let priv = 'normal';
		if (common.checkAuth('moderator', client.ident))
			priv = 'mod';
		else if (common.checkAuth('janitor', client.ident))
			priv = 'janny';

		const handler = msg => client.send(msg);
		this.on(priv, handler);
		client.once('close', () =>
			this.removeListener(priv, handler).checkCount());
	}
	checkCount() {
		if (this.countListeners())
			return;
		if (this.idleOutTimer)
			clearTimeout(this.idleOutTimer);
		this.idleOutTimer = setTimeout(() =>
			!this.countListeners() && this.commitSudoku(), 30000);
	}
	countListeners() {
		let count = 0;
		for (let priv of ['normal', 'janny', 'mod']) {
			count += this.listenerCount(priv);
		}
		return count;
	}
	onMessage(unparsed) {
		const [msg, extra] = JSON.parse(unparsed);
		this.emit('normal', msg);

		// Bundle additional info for priveledged clients
		for (let priv of ['janny', 'mod']) {
			const extended = extra && extra[priv] && msg.concat([extra[priv]]);
			this.emit(priv, extended || msg);
		}
	}
}

// Can't define static properties with ES6 classes. ES7 when?
Subscription.keys = {};
module.exports = Subscription;
