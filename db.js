/*
Main redis controller module
 */

const _ = require('underscore'),
    async = require('async'),
    cache = require('./server/state').dbCache,
    caps = require('./server/caps'),
    common = require('./common'),
    config = require('./config'),
    events = require('events'),
    fs = require('fs'),
    hooks = require('./util/hooks'),
    hot = require('./server/state').hot,
	okyaku = require('./server/okyaku'),
    Muggle = require('./util/etc').Muggle,
	nodeRedis = require('redis'),
    tail = require('./util/tail'),
    winston = require('winston');

//Caches. Maps post numbers to threads.
const OPs = exports.OPs = cache.OPs,
	// Maps thread numbers to boards
	BOARDS = exports.boards = cache.boards,
	// Contains all public redis subscriptions for sharing
	SUBS = exports.SUBS = cache.threadSubs;

function redis_client() {
	const client = nodeRedis.createClient(config.REDIS_PORT);
	client.select(config.redis_database || 0);
	return client;
}
exports.redis_client = redis_client;
const redis = global.redis = redis_client();
redis.on('error', err => winston.error('Redis error:', err));

// Validate database spec version
{
	const dbVersion = '1',
		m = redis.multi();
	m.get('postctr');
	m.get('dbVersion');
	m.exec((err, res) => {
		if (err)
			throw err;

		// If post counter does not exist, we assume the database does not
		// have any posts
		if (!res[0]) {
			redis.set('dbVersion', dbVersion, err => {
				if (err)
					throw err;
			});
		}
		else if (res[1] !== dbVersion) {
			throw new Error(`Incompatable database version: ${res[1] || 0}: `
				+ 'See docs/migration.md')
		}
	})
}

// Depend on global redis client
const admin = require('./server/admin'),
	amusement = require('./server/amusement'),
	imager = require('./imager');

exports.UPKEEP_IDENT = {
	auth: 'Upkeep',
	ip: '127.0.0.1'
};

/* REAL-TIME UPDATES */

class Subscription extends events.EventEmitter {
	constructor(targetInfo) {
		super();
		this.setMaxListeners(0);

		this.fullKey = targetInfo.key;
		this.target = targetInfo.target;
		this.channel = targetInfo.channel;
		SUBS[this.fullKey] = this;

		this.pending_subscriptions = [];
		this.subscription_callbacks = [];

		this.k = redis_client();
		this.k.on('error', this.on_sub_error.bind(this));
		this.k.on('subscribe', this.on_one_sub.bind(this));
		this.k.subscribe(this.target);
		this.pending_subscriptions.push(this.target);
		if (this.target != this.fullKey) {
			this.k.subscribe(this.fullKey);
			this.pending_subscriptions.push(this.fullKey);
		}
	}
	on_one_sub(name) {
		const i = this.pending_subscriptions.indexOf(name);
		if (i < 0)
			throw "Obtained unasked-for subscription " + name + "?!";
		this.pending_subscriptions.splice(i, 1);
		if (this.pending_subscriptions.length == 0)
			this.on_all_subs();
	}
	on_all_subs() {
		let k = this.k;
		k.removeAllListeners('subscribe');
		k.on('message', this.on_message.bind(this));
		k.removeAllListeners('error');
		k.on('error', this.sink_sub.bind(this));
		this.subscription_callbacks.forEach(sub => sub(null));
		delete this.pending_subscriptions;
		delete this.subscription_callbacks;
	}
	sink_sub(err) {
		if (config.DEBUG)
			throw err;
		this.emit('error', this.target, err);
		this.commit_sudoku();
	}
	commit_sudoku() {
		const k = this.k;
		k.removeAllListeners('error');
		k.removeAllListeners('message');
		k.removeAllListeners('subscribe');
		k.quit();
		if (SUBS[this.fullKey] === this)
			delete SUBS[this.fullKey];
		this.removeAllListeners('update');
		this.removeAllListeners('error');
	}
	on_sub_error(err) {
		winston.error("Subscription error:", (err.stack || err));
		this.commit_sudoku();
		this.subscription_callbacks.forEach(sub => sub(err));
		this.subscription_callbacks = null;
	}
	on_message(chan, msg) {
		/* Do we need to clarify whether this came from target or fullKey? */
		const parsed = this.parse_pub_message(msg);
		let extra;
		if (this.channel && parsed.suffixPos) {
			const suffix = JSON.parse(msg.slice(parsed.suffixPos));
			extra = suffix[this.channel];
		}
		msg = parsed.body;
		const m = msg.match(/^(\d+),(\d+)/),
			op = parseInt(m[1], 10),
			kind = parseInt(m[2], 10);

		if (extra) {
			const modified = this.inject_extra(kind, msg, extra);
			// currently this won't modify op or kind,
			// but will have to watch out for that if that changes
			if (modified)
				msg = modified;
		}
		this.emit('update', op, kind, '[[' + msg + ']]');
	}
	parse_pub_message(msg) {
		const m = msg.match(/^(\d+)\|/),
			prefixLen = m[0].length,
			bodyLen = parseInt(m[1], 10),
			suffixPos = prefixLen + bodyLen,
			info = {body: msg.substr(prefixLen, bodyLen)};
		if (msg.length > suffixPos)
			info.suffixPos = suffixPos;
		return info;
	}
	inject_extra(kind, msg, extra) {
		// XXX: Why the fuck don't you just stringify arrays?
		const parsed = JSON.parse(`[${msg}]`);
		switch (kind) {
			case common.INSERT_POST:
				parsed[2].mnemonic = extra.mnemonic;
				break;

			// Add moderation information for staff
			case common.SPOILER_IMAGES:
			case common.DELETE_IMAGES:
			case common.DELETE_POSTS:
			case common.LOCK_THREAD:
			case common.UNLOCK_THREAD:
			case common.BAN:
				parsed.push(extra);
				break;
			default:
				return;
		}
		return JSON.stringify(parsed).slice(1, -1);
	}
	has_no_listeners() {
		/* Possibly idle out after a while */
		if (this.idleOutTimer)
			clearTimeout(this.idleOutTimer);
		this.idleOutTimer = setTimeout(() => {
			this.idleOutTimer = null;
			if (this.listeners('update').length == 0)
				this.commit_sudoku();
		}, 30000);
	}
	static get(target, ident) {
		const full = Subscription.full_key(target, ident);
		return SUBS[full.key] || new Subscription(full);
	}
	static full_key(target, ident) {
		let channel;
		if (common.checkAuth('dj', ident))
			channel = 'mod';
		else if (common.checkAuth('janitor', ident))
			channel = 'auth';
		const key = channel ? `${channel}:${target}` : target;
		return {key, channel, target};
	}
	when_ready(cb) {
		if (this.subscription_callbacks)
			this.subscription_callbacks.push(cb);
		else
			cb(null);
	}
}

/* OP CACHE */

// Validate the post number points to a thread on the specific board
function validateOP(op, board) {
	return OPs[op] == op && BOARDS[op] === board;
}
exports.validateOP = validateOP;

function track_OPs (callback) {
	const redis = redis_client();
	redis.subscribe('cache');
	redis.once('subscribe', () => load_OPs(callback));
	redis.on('message', update_cache);
	// connection persists for the purpose of cache updates
}
exports.track_OPs = track_OPs;

// Build caches on server start
function load_OPs(callback) {
	const boards = config.BOARDS,
		threadList = [];
	async.waterfall([
		next => {
			const m = redis.multi();
			for (let board of boards) {
				m.zrange(`board:${board}:threads`, 0, -1);
			}
			m.exec(next);
		},
		(allThreads, next) => {
			// Gather posts from all boards and their threads
			const m = redis.multi();
			for (let i = 0; i < boards.length; i++) {
				for (let thread of allThreads[i]) {
					const num = parseInt(thread, 10);
					cacheBoard(boards[i], num);

					// For consitency, an OP will have it's own post
					// number in the cache
					cacheOP(num, num);
					threadList.push(num);
					m.lrange(`thread:${thread}:posts`, 0, -1);
				}
			}
			m.exec(next);
		},
		(threads, next) => {
			for (let i = 0; i < threads.length; i++) {
				for (let post of threads[i]) {
					cacheOP(post, threadList[i]);
				}
			}
			next();
		}
	], callback);
}

function cacheBoard(board, op) {
	BOARDS[op] = board;
}

function cacheOP(num, op) {
	OPs[num] = op;
}

function uncacheThread(num) {
	delete BOARDS[num];

	// And all of the thread's posts. This includes the OP.
	for (let post in OPs) {
		if (OPs[post] == num)
			delete OPs[post];
	}
}

// To have as little inconsitent state between the database and server(s)
// very little is chached into memory - only thread to board and post to
// thread parenthood. This function is called through redis pub/sun to ensure
// cache consitency between possible  multiple slave servers, once
// clustering is implemnted.
function update_cache(chan, msg) {
	msg = JSON.parse(msg);
	let [kind, num, parent] = msg;
	switch (kind) {
		// Insert thread
		case 0:
			cacheBoard(parent, num);
			parent = num;
		// Insert post
		case 1:
			cacheOP(num, parent);
			break;
		// Thread purged from database
		case 2:
			uncacheThread(num);
			break;
		// Reaload ban list and reverify all connected clients
		case 3:
			admin.loadBans(okyaku.scan_client_caps);
			break;
	}
}

function on_pub (name, handler) {
	const redis = redis_client();
	redis.subscribe(name);
	redis.on('message', handler);
}
exports.on_pub = on_pub;

/* SOCIETY */

// Main database controller class
class Yakusoku extends events.EventEmitter {
	constructor(board, ident) {
		super();
		this.id = ++(cache.YAKUMAN);
		this.board = board;

		//Should moderation be allowed on this board?
		this.isContainmentBoard	= config.containment_boards.indexOf(board) > -1;
		this.ident = ident;
		this.subs = [];
	}
	disconnect() {
		this.removeAllListeners();
	}
	kiku(targets, on_update, on_sink, callback) {
		this.on_update = on_update;
		this.on_sink = on_sink;
		forEachInObject(targets, (id, cb) => {
			const target = this.target_key(id),
				sub = Subscription.get(target, this.ident);
			sub.on('update', on_update);
			sub.on('error', on_sink);
			this.subs.push(sub.fullKey);
			sub.when_ready(cb);
		}, callback);
	}
	target_key(id) {
		return id === 'live' ? 'board:' + this.board : 'thread:' + id;
	}
	kikanai() {
		for (let i = 0; i < this.subs.length; i++) {
			const sub = SUBS[this.subs[i]];
			if (!sub)
				continue;
			sub.removeListener('update', this.on_update);
			sub.removeListener('error', this.on_sink);
			if (sub.listeners('update').length == 0)
				sub.has_no_listeners();
		}
		this.subs = [];
		return this;
	}
	reserve_post(op, ip, callback) {
		if (config.READ_ONLY)
			return callback(Muggle("Can't post right now."));
		if (ip === '127.0.0.1')
			return this.reserve(op, callback);

		const key = `ip:${ip}:throttle:`,
			now = Date.now(),
			shortTerm = key + this.short_term_timeslot(now),
			longTerm = key + this.long_term_timeslot(now);
		redis.mget([shortTerm, longTerm], (err, quants) => {
			if (err)
				return callback(Muggle("Limiter failure.", err));
			if (quants[0] > config.SHORT_TERM_LIMIT
				|| quants[1] > config.LONG_TERM_LIMIT
			)
				return callback(Muggle('Reduce your speed.'));

			this.reserve(op, callback);
		});
	}
	reserve(op, cb) {
		redis.incr('postctr', function(err, num) {
			if (err)
				return cb(err);
			OPs[num] = op || num;
			cb(null, num);
		});
	}
	short_term_timeslot(when) {
		return Math.floor(when / (1000 * 60 * 5));
	}
	long_term_timeslot(when) {
		return Math.floor(when / (1000 * 60 * 60 * 24));
	}
	insert_post(msg, body, extra, callback) {
		if (config.READ_ONLY)
			return callback(Muggle("Can't post right now."));
		if (!this.board)
			return callback(Muggle("Can't retrieve board for posting."));
		const {num} = msg,
			op = msg.op || num,
			{ip, board} = extra,
			isThead = !msg.op;
		if (!num)
			return callback(Muggle("No post number."));
		else if (!ip)
			return callback(Muggle("No IP."));
		else if (!isThead && !validateOP(op, board)) {
			uncacheThread(op);
			return callback(Muggle('Thread does not exist.'));
		}

		const view = {
			time: msg.time,
			num,
			board,
			ip,
			state: msg.state.join()
		};
		const optPostFields = ['name', 'trip', 'email', 'auth', 'subject',
			'dice'];
		for (let field of optPostFields) {
			if (msg[field])
				view[field] = msg[field];
		}
		const boardKey = 'board:' + this.board;
		if (!isThead)
			view.op = op;

		if (extra.image_alloc) {
			msg.image = extra.image_alloc.image;
			if (isThead == msg.image.pinky)
				return callback(Muggle("Image is the wrong size."));
			delete msg.image.pinky;
		}

		const key = (isThead ? 'thread:' : 'post:') + num,
			m = redis.multi();
		m.incr(boardKey + ':postctr'); // must be first
		m.sadd('liveposts', key);

		hooks.trigger_sync('inlinePost', {
			src: msg,
			dest: view
		});

		if (msg.image) {
			if (isThead)
				view.imgctr = 1;
			else
				m.hincrby('thread:' + op, 'imgctr', 1);
			this.imageDuplicateHash(m, msg.image.hash, msg.num);
		}
		m.hmset(key, view);
		m.set(key + ':body', body);

		const dice = msg.dice;
		if (dice) {
			this.writeDice(m, dice, key);
			view.dice = dice;
		}
		const links = msg.links;
		if (links) {
			m.hmset(key + ':links', links);
			view.links = links;
			this.addBacklinks(m, num, op, links);
		}

		const etc = {augments: {}};
		if (isThead) {
			etc.cacheUpdate = [0, num, board];
			/* Rate-limit new threads */
			if (~['127.0.0.1', '::1'].indexOf(ip))
				m.setex(`ip:${ip}:throttle:thread`, config.THREAD_THROTTLE, op);
		}
		else {
			etc.cacheUpdate = [1, num, op];
			m.rpush(`thread:${op}:posts`, num);
		}

		/* Denormalize for backlog */
		view.nonce = msg.nonce;
		view.body = body;

		let bump;
		async.waterfall(
			[
				function (next) {
					if (!msg.image)
						return next();
					imager.commit_image_alloc(extra.image_alloc, next);
				},
				// Determine, if we need to bump the thread to the top of
				// the board
				function(next) {
					if (isThead) {
						bump = true;
						return next();
					}

					redis.llen(`thread:${op}:posts`, function(err, res) {
						if (err)
							return next(err);
						bump = !common.is_sage(view.email)
							&& res < config.BUMP_LIMIT[board];
						next();
					});
				},
				next => {
					if (ip) {
						const n = this.post_volume(view, body);
						if (n > 0)
							this.update_throughput(m, ip, view.time, n);

						// Only the client-private Reader() instances need
						// to embed mnemonics in-post. Doing that here would
						// publish it to everyone. Instead live mnemonic
						// updates are pushed through the 'mod' channel to
						// authenticated moderatots and up only.
						const mnemonic = admin.genMnemonic(ip);
						if (mnemonic)
							etc.augments.mod = {mnemonic};
					}

					// Don't parse dice, because they aren't stringified on
					// live publishes
					extract(view);
					if (bump)
						m.incr(boardKey + ':bumpctr');
					this._log(m, op, common.INSERT_POST, [view, bump], etc);
					m.exec(next);
				},
				function(res, next) {
					if (!bump)
						return next();
					redis.zadd(boardKey + ':threads', res[0], op, next);
				}
			],
			function (err) {
				if (err) {
					delete OPs[num];
					return callback(err);
				}
				callback();
			}
		);
	}
	imageDuplicateHash(m, hash, num) {
		m.zadd('imageDups', Date.now() + (config.DEBUG ? 30000 : 3600000),
			`${num}:${hash}`);
	}
	writeDice(m, dice, key) {
		dice.forEach(die => m.lpush(key + ':dice', JSON.stringify(die)))
	}
	addBacklinks(m, num, op, links) {
		for (let targetNum in links) {
			// Check if post exists through cache
			if (!(targetNum in OPs))
				continue;
			const key = (targetNum in BOARDS ? 'thread' : 'post')
				+ `:${targetNum}:backlinks`;
			m.hset(key, num, op);
			this._log(m, links[targetNum], common.BACKLINK,
				[targetNum, num, op]);
		}
	}
	post_volume(view, body) {
		return (body ? body.length : 0) +
			(view ? (config.NEW_POST_WORTH || 0) : 0) +
			((view && view.image) ? (config.IMAGE_WORTH || 0) : 0);
	}
	update_throughput(m, ip, when, quant) {
		const key = `ip:${ip}:throttle:`,
			shortKey = key + this.short_term_timeslot(when),
			longKey = key + this.long_term_timeslot(when);
		m.incrby(shortKey, quant);
		m.incrby(longKey, quant);

		/* Don't want to use expireat in case of timezone trickery
		 or something dumb. (Really, UTC should be OK though...) */
		// Conservative expirations
		m.expire(shortKey, 10 * 60);
		m.expire(longKey, 2 * 24 * 3600);
	}
	_log(m, op, kind, msg, opts = {}) {
		msg = JSON.stringify(msg).slice(1, -1);
		msg = msg.length ? (kind + ',' + msg) : ('' + kind);
		winston.verbose("Log: " + msg);
		if (!op)
			throw new Error('No OP.');
		const key = 'thread:' + op;

		if (common.is_pubsub(kind)) {
			m.rpush(key + ':history', msg);
			m.hincrby(key, 'hctr', 1);
		}

		const opBit = op + ',',
			len = opBit.length + msg.length;
		msg = len + '|' + opBit + msg;

		if (!_.isEmpty(opts.augments))
			msg += JSON.stringify(opts.augments);
		m.publish(key, msg);
		m.publish('board:' + this.board, msg);
		if (opts.cacheUpdate)
			m.publish('cache', JSON.stringify(opts.cacheUpdate));
	}
	add_image(post, alloc, ip, callback) {
		const {num, op} = post;
		if (!op)
			return callback(Muggle("Can't add another image to an OP."));
		const {image} = alloc;
		if (!image.pinky)
			return callback(Muggle("Image is wrong size."));
		delete image.pinky;

		const key = 'post:' + num;
		async.waterfall([
			next => redis.exists(key, next),
			(exists, next) => {
				if (!exists)
					return next(Muggle("Post does not exist."));
				imager.commit_image_alloc(alloc, next);
			},
			next => {
				const m = redis.multi();
				this.imageDuplicateHash(m, image.hash, num);
				m.hmset(key, image);
				m.hincrby('thread:' + op, 'imgctr', 1);

				// Useless once image is commited
				delete image.hash;
				this._log(m, op, common.INSERT_IMAGE, [num, image]);

				const now = Date.now();
				this.update_throughput(m, ip, now,
					this.post_volume({image: true}));
				m.exec(next);
			}
		], callback);
	}
	append_post(post, tail, old_state, extra, cb) {
		const m = redis.multi(),
			key = (post.op ? 'post:' : 'thread:') + post.num;

		/* Don't need to check .exists() thanks to client state */
		m.append(key + ':body', tail);

		/* XXX: fragile */
		if (old_state[0] != post.state[0] || old_state[1] != post.state[1])
			m.hset(key, 'state', post.state.join());
		if (extra.ip) {
			const now = Date.now();
			this.update_throughput(m, extra.ip, now,
				this.post_volume(null, tail));
		}
		if (!_.isEmpty(extra.new_links))
			m.hmset(key + ':links', extra.new_links);

		const {num} = post,
			op = post.op || num,

		// TODO: Make less dirty, when post state is refactored
			_extra = {state: [old_state[0] || 0, old_state[1] || 0]};
		const {links} = extra;
		if (links) {
			_extra.links = links;
			this.addBacklinks(m, num, op, links);
		}
		const {dice} = extra;
		if (dice) {
			_extra.dice = dice;
			this.writeDice(m, dice, key);
		}

		this._log(m, op, common.UPDATE_POST, [num, tail, _extra]);
		m.exec(cb);
	}
	finish_post(post, callback) {
		const m = redis.multi(),
			key = (post.op ? 'post:' : 'thread:') + post.num;

		/* Don't need to check .exists() thanks to client state */
		this.finish_off(m, key, post.body);
		this._log(m, post.op || post.num, common.FINISH_POST, [post.num]);
		m.exec(callback);
	}
	finish_off(m, key, body) {
		m.hset(key, 'body', body);
		m.del(key + ':body');
		m.hdel(key, 'state');
		m.srem('liveposts', key);
	}
	finish_all(callback) {
		redis.smembers('liveposts', (err, keys) => {
			if (err)
				return callback(err);
			async.forEach(keys, (key, cb) => {
				const m = redis.multi();
				m.get(key + ':body');
				const isPost = key.slice(0, 5) == 'post:';
				if (isPost)
					m.hget(key, 'op');
				m.exec((err, rs) => {
					if (err)
						return cb(err);
					const m = redis.multi();
					this.finish_off(m, key, rs[0]);
					const n = parseInt(key.match(/:(\d+)$/)[1]),
						op = isPost ? parseInt(rs[1], 10) : n;
					this._log(m, op, common.FINISH_POST, [n]);
					m.srem('liveposts', key);
					m.exec(cb);
				});
			}, callback);
		});
	}
	fetch_backlogs(watching, callback) {
		const combined = [];
		forEachInObject(watching,
			function (thread, cb) {
				if (thread == 'live')
					return cb();
				const key = `thread:${thread}:history`,
					sync = watching[thread];
				redis.lrange(key, sync, -1, function (err, log) {
					if (err)
						return cb(err);
					const prefix = thread + ',';
					for (let i = 0; i < log.length; i++) {
						combined.push(prefix + log[i]);
					}
					cb(null);
				});
			},
			function (errs) {
				callback(errs, combined);
			}
		);
	}
	check_thread_locked(op, callback) {
		redis.hexists('thread:' + op, 'locked', function(err, lock) {
			if (err)
				return callback(err);
			callback(lock ? Muggle('Thread is locked.') : null);
		});
	}
	check_throttle(ip, callback) {
		// So we can spam new threads in debug mode
		if (config.DEBUG)
			return callback(null);
		redis.exists(`ip:${ip}:throttle:thread`, function(err, exists) {
			if (err)
				return callback(err);
			callback(exists ? Muggle('Too soon.') : null);
		});
	}
	get_tag(page) {
		const keyBase = 'board:' + this.board,
			key = keyBase + ':threads',
			// -1 is for live pages and -2 is for catalog
			catalog = page === -2;
		if (page < 0)
			page = 0;
		let start, end;
		if (catalog) {
			// Read all threads
			start = 0;
			end = -1;
		}
		else {
			start = page * hot.THREADS_PER_PAGE;
			end = start + hot.THREADS_PER_PAGE - 1;
		}

		const m = redis.multi();
		m.zrevrange(key, start, end);
		m.zcard(key);

		// Used for building board eTags
		m.get(keyBase + ':postctr');
		m.exec((err, res) => {
			if (err)
				return this.emit('error', err);
			const nums = res[0];
			if (page > 0 && !nums.length)
				return this.emit('nomatch');
			this.emit('begin', res[1] || 0, res[2] || 0);
			const reader = new Reader(this.ident);

			// Proxy Reader events to Yakusoku
			reader.on('error', this.emit.bind(this, 'error'));
			reader.on('thread', this.emit.bind(this, 'thread'));
			reader.on('post', this.emit.bind(this, 'post'));
			reader.on('endthread', this.emit.bind(this, 'endthread'));
			this._get_each_thread(reader, 0, nums, catalog);
		});
	}
	_get_each_thread(reader, ix, nums, catalog) {
		if (!nums || ix >= nums.length) {
			this.emit('end');
			reader.removeAllListeners('endthread');
			reader.removeAllListeners('end');
			return;
		}

		const self = this;
		function next_please() {
			reader.removeListener('end', next_please);
			reader.removeListener('nomatch', next_please);
			self._get_each_thread(reader, ix+1, nums, catalog);
		}

		reader.on('end', next_please);
		reader.on('nomatch', next_please);
		reader.get_thread(nums[ix], {
			catalog,
			abbrev: hot.ABBREVIATED_REPLIES || 5
		});
	}
	// Purges all the thread's keys from the database and delete's all images
	// contained
	purge_thread(op, board, callback) {
		const key = 'thread:' + op,
			// Key suffixes that might or might not exist
			optional = ['links', 'backlinks', 'body', 'dice', 'mod'],
			keysToDel = [],
			filesToDel = [],
			nums = [];
		async.waterfall([
			// Confirm thread can be deleted
			next => redis.exists(key, next),
			(res, next) => {
				// Likely to happen, if interrupted mid-purge
				if (!res) {
					redis.zrem(`board:${board}:threads`, op);
					return callback();
				}

				// Get reply list
				redis.lrange(key + ':posts', 0, -1, next);
			},
			// Read all post hashes
			(posts, next) => {
				const m = redis.multi();
				for (let i = 0; i < posts.length; i++) {
					// Queue for removal from post cache
					nums.push(posts[i]);
					posts[i] = 'post:' + posts[i];
				}

				// Parse OP key like all other hashes. `res` will always be an
				// array, even if empty.
				posts.unshift(key);
				for (let key of posts) {
					m.hgetall(key);
					for (let suffix of optional) {
						m.exists(`${key}:${suffix}`);
					}
				}

				// A bit more complicated, because we need to pass two arguments
				// to the next function, to map the arrays
				m.exec((err, res) => next(err, res, posts));
			},
			// Populate key and file to delete arrays
			(res, posts, next) => {
				const imageTypes = ['src', 'thumb', 'mid'];
				for (let i = 0; i < res.length; i += 6) {
					const hash = res[i],
						key = posts[i / 6];
					if (!hash)
						continue;

					keysToDel.push(key);
					for (let o = 0; o < optional.length; o++) {
						if (res[i + o])
							keysToDel.push(`${key}:${optional[o]}`);
					}

					// Add images to delete list
					for (let type of imageTypes) {
						const image = hash[type];
						if (image)
							filesToDel.push(imager.media_path(type, image));
					}
				}
				next();
			},
			// Check for OP-only keys
			next => {
				const m = redis.multi(),
					OPKeys = [];
				for (let suffix of ['history', 'posts']) {
					OPKeys.push(`${key}:${suffix}`);
				}
				for (let key of OPKeys) {
					m.exists(key);
				}
				m.exec((err, res) => next(err, res, OPKeys));
			},
			(res, OPKeys, next) => {
				for (let i = 0; i < res.length; i++) {
					if (res[i])
						keysToDel.push(OPKeys[i]);
				}

				// Delete all keys
				const m = redis.multi();
				for (let key of keysToDel) {
					m.del(key);
				}
				m.exec(next);
			},
			(res, next) =>
				// Delete all images
				async.each(filesToDel,
					(file, cb) =>
						fs.unlink(file, err => cb(err)),
					err => next(err)),
			next => {
				const m = redis.multi();
				m.zrem(`board:${board}:threads`, op, next);
				// Clear thread and post numbers from caches on all slaves
				m.publish('cache', JSON.stringify([2, +op]));
				m.exec(next);
			}
		], callback);
	}
	get_banner(cb) {
		redis.get('banner:info', cb);
	}
	set_banner(message, cb) {
		redis.set('banner:info', message, err => {
			if (err)
				return cb(err);

			// Dispatch new banner
			const m = redis.multi();
			this._log(m, 0, common.UPDATE_BANNER, [message]);
			m.exec(cb);
		});
	}
	modHandler(kind, nums, cb) {
		if (this.isContainmentBoard)
			return false;

		// Group posts by thread for live publishes to the clients
		const threads = {};
		for (let num of nums) {
			const op = OPs[num];
			if (!(op in threads))
				threads[op] = [];
			threads[op].push(num);
		}
		async.forEachOf(threads, (nums, op, cb) =>
			this.handleModeration(nums, op, kind, cb),
		cb);
		return true;
	}
	handleModeration(nums, op, kind, cb) {
		const opts = this.moderationSpecs[kind],
			{props, check, persist} = opts,
			keys = [];
		async.waterfall([
			// Read required post properties from redis
			next => {
				const m = redis.multi();
				for (let num of nums) {
					const key = postKey(num, op);
					keys.push(key);
					const command = props.slice();
					command.unshift(key);
					m.hmget(command);
				}
				m.exec(next);
			},
			(res, next) => {
				const m = redis.multi();
				for (let i = 0; i < res.length; i++) {
					// Check if post is eligible for moderation action
					if (check(res[i]))
						continue;

					// Persist to redis
					const key = keys[i],
						num = nums[i],
						msg = [num];
					persist(m, key, msg);

					// Live publish
					this.logModeration(m, {key, op, kind, num, msg});
				}
				m.exec(next);
			}
		], cb);
	}
	logModeration(m, opts) {
		const time = Date.now();
		const info = {
			time,
			num: opts.num,
			op: opts.op,
			ident: this.hideEmail(),
			kind: opts.kind
		};

		const stringified = JSON.stringify(info);
		m.lpush(opts.key + ':mod', stringified);
		m.zadd('modLog', time, stringified);

		this._log(m, opts.op, opts.kind, opts.msg, {
			augments: {
				// Duplicating channels for now. Will add some differences later
				auth: info,
				mod: info
			}
		});
	}
	// Abstract the email as to not reveal it to all staff
	hideEmail() {
		return config.staff[this.ident.auth][this.ident.email];
	}
	// Bans are somewhat more complicated and do not fit into the common
	// modHandler() pathway. Plenty of duplication here, because of that.
	ban(msg, cb) {
		const [num, days, hours, minutes, reason, display] = msg,
			now = Date.now(),
			till = ((days * 24 + hours) * 60 + minutes) * 60 * 1000 + now,
			op = OPs[num],
			key = `${op === num ? 'thread' : 'post'}:${num}`;
		async.waterfall([
			next => redis.hget(key, 'ip', next),
			(ip, next) => {
				const info = {
					num, op, till, reason,
					time: now,
					ident: this.hideEmail(),
					kind: common.BAN
				};

				// Publically display a ban to all users, if display option
				// set. Otherwise only published to staff and others get a
				// useless 0, for compatability with the current pub/sub
				// pathway.
				const m = redis.multi();
				this._log(m, op, common.BAN, [display ? num : 0], {
					augments: {
						auth: info,
						mod: info
					},
					cacheUpdate: [3]
				});

				m.lpush(key + ':mod', JSON.stringify(info));

				// Mnemonic needed only for logging
				info.mnemonic = admin.genMnemonic(ip);
				m.zadd('modLog', now, JSON.stringify(info));

				// XXX: The ban duration in the sorted set will that of the
				// most recently applied ban to this IP.
				m.zadd('bans', till, ip);
				if (display)
					m.hset(key, 'banned', 1);
				m.exec(next);
			}
		], cb);
	}
	// We don't pass the banned IPs to clients, so now we have to fetch all
	// the banned IPs, generate a mnemonic for each and remove the
	// corresponding one, if any.
	unban(mnemonic, cb) {
		async.waterfall([
			next => redis.zrange('bans', 0, -1, next),
			(bans, next) => {
				let match;
				for (let ip of bans) {
					if (admin.genMnemonic(ip) === mnemonic) {
						match = ip;
						break;
					}
				}

				if (!match)
					return cb();
				const m = redis.multi();
				m.zrem('bans', match);
				m.publish('cache', '[3]');

				const time = Date.now();
				m.zadd('modLog', time, JSON.stringify({
					time,
					ident: this.hideEmail(),
					kind: common.UNBAN
				}));

				m.exec(next);
			}
		], cb);
	}
}

// Options for various moderation actions. No class properties in ES6, so
// keep them here.
Yakusoku.prototype.moderationSpecs = {
	[common.SPOILER_IMAGES]: {
		props: ['src', 'spoiler'],
		check(res) {
			// No image or already spoilt
			return !res[0] || !!res[1];
		},
		persist(m, key, msg) {
			const spoiler = common.pick_spoiler(-1).index;
			m.hset(key, 'spoiler', spoiler);
			msg.push(spoiler);
		}
	},
	[common.DELETE_IMAGES]: {
		props: ['src', 'imgDeleted'],
		check(res) {
			// No image or already hidden
			return !res[0] || !!res[1];
		},
		persist(m, key) {
			m.hset(key, 'imgDeleted', 1);
		}
	},
	[common.DELETE_POSTS]: {
		props: ['deleted'],
		check(res) {
			return !!res[0];
		},
		persist(m, key) {
			m.hset(key, 'deleted', 1);
		}
	},
	[common.LOCK_THREAD]: {
		props: ['locked'],
		check(res) {
			return !!res[0];
		},
		persist(m, key) {
			m.hset(key, 'locked', 1);
		}
	},
	[common.UNLOCK_THREAD]: {
		props: ['locked'],
		check(res) {
			return !res[0];
		},
		persist(m, key) {
			m.hdel(key, 'locked');
		}
	}
};

exports.Yakusoku = Yakusoku;

/* LURKERS */

class Reader extends events.EventEmitter {
	constructor(ident) {
		// Call the EventEmitter's constructor
		super()
        this.canSeeMnemonics = ident.auth === 'dj'
            || common.checkAuth('moderator', ident)
        this.canSeeModeration = common.checkAuth('janitor', ident)
	}
	get_thread(num, opts) {
		const key = 'thread:' + num;
		redis.hgetall(key, (err, pre_post) => {
			if (err)
				return this.emit('error', err);
			if (!pre_post || !this.formatPost(pre_post))
				return this.emit('nomatch');

			this.emit('begin', pre_post);

			let nums, opPost,
				total = 0;
			const abbrev = opts.abbrev || 0;
			async.waterfall(
				[
					next => this.with_body(key, pre_post, next),
					(fullPost, next) => {
						opPost = fullPost;
						const m = redis.multi(),
							postsKey = key + ':posts';

						// order is important!
						m.lrange(postsKey, -abbrev, -1);

						// The length of the above array is limited by the
						// amount of posts we are retrieving. A total number
						// of posts is quite useful.
						m.llen(postsKey);
						this.getExtras(m, key);
						if (abbrev)
							m.llen(postsKey);
						m.exec(next);
					},
					(rs, next) => {
						// get results in the same order as before
						nums = rs.shift();
						// NOTE: these are only the displayed replies, not
						// all of them
						opPost.replies = nums || [];
						opPost.replyctr = parseInt(rs.shift(), 10) || 0;
						this.parseExtras(rs, opPost);
						if (abbrev)
							total += parseInt(rs.shift(), 10);

						opPost.omit = Math.max(total - abbrev, 0);
						opPost.hctr = parseInt(opPost.hctr, 10);

						// So we can pass a thread number on `endthread`
						// emission
						opts.op = opPost.num;
						next();
					}
				],
				err => {
					if (err)
						return this.emit('error', err);
					this.emit('thread', opPost);
					if (opts.catalog)
						return this.emit('end');
					this._get_each_reply(0, nums, opts);
				}
			);
		});
	}
	getExtras(m, key) {
		m.hgetall(key + ':links');
		m.hgetall(key + ':backlinks');
		m.lrange(key + ':dice', 0, -1);
		if (this.canSeeModeration)
			m.lrange(key + ':mod', 0, -1);
	}
	parseExtras(res, post) {
		for (let key of ['links', 'backlinks']) {
			const prop = res.shift();
			if (prop)
				post[key] = prop;
		}
		this.parseStringList(res, 'dice', post);
		if (this.canSeeModeration)
			this.parseStringList(res, 'mod', post);
	}
	parseStringList(res, prop, post) {
		const list = res.shift();
		if (!list.length)
			return;

		// Reverse to preserve chronological order
		post[prop] = destringifyList(list.reverse())
	}
	formatPost(post) {
		if (!this.canSeeModeration) {
			if (post.deleted)
				return false;
			if (post.imgDeleted)
				imager.deleteImageProps(post);
		}
		if (this.canSeeMnemonics) {
			const mnemonic = admin.genMnemonic(post.ip);
			if (mnemonic)
				post.mnemonic = mnemonic;
		}
		extract(post);
		return true;
	}
	_get_each_reply(ix, nums, opts) {
		if (!nums || ix >= nums.length) {
			this.emit('endthread', opts.op);
			this.emit('end');
			return;
		}
		const num = parseInt(nums[ix], 10);
		this.get_post('post', num, (err, post) => {
			if (err)
				return this.emit('error', err);
			if (post)
				this.emit('post', post);
			this._get_each_reply(ix + 1, nums, opts);
		});
	}
	get_post(kind, num, cb) {
		const key = `${kind}:${num}`;
		async.waterfall([
			next => {
				const m = redis.multi();
				m.hgetall(key);
				this.getExtras(m, key);
				m.exec(next);
			},
			(data, next) => {
				const pre_post = data.shift();
				this.parseExtras(data, pre_post);
				if (_.isEmpty(pre_post))
					return next();

				pre_post.num = num;
				if (kind === 'post')
					pre_post.op = parseInt(pre_post.op, 10);
				this.with_body(key, pre_post, next);
			},
			(post, next) => {
				if (post) {
					if (!this.formatPost(post))
						post = null;
				}
				next(null, post);
			}
		],	cb);
	}
	with_body(key, post, callback) {
		if (post.body !== undefined)
			return callback(null, post);

		redis.get(key + ':body', function(err, body) {
			if (err)
				return callback(err);
			if (body !== null) {
				post.body = body;
				post.editing = true;
				return callback(null, post);
			}

			// Race condition between finishing posts
			redis.hget(key, 'body', function(err, body) {
				if (err)
					return callback(err);
				post.body = body;
				callback(null, post);
			});
		});
	}
	// Wrapper for retrieving individual posts separatly from threads
	singlePost(num, ident, cb) {
		const info = postInfo(num),
			key = info.isOP ? 'thread' : 'post';
		if (!caps.can_access_board(ident, info.board))
			return cb();
		this.get_post(key, num, function(err, post) {
			if (err || !post)
				return cb();
			cb(post);
		})
	}
}
exports.Reader = Reader;

// Retrieve post info from cache
function postInfo(num) {
	const isOP = num in BOARDS;
	return {
		isOP,
		board: isOP ? BOARDS[num] : BOARDS[OPs[num]]
	};
}
exports.postInfo = postInfo;

/* HELPERS */

function is_board (board) {
	return config.BOARDS.indexOf(board) >= 0;
}
exports.is_board = is_board;

// Format post hash for passing to renderer and clients
function extract(post) {
	// Only used internally and should not be exported to clients
	for (let key of ['ip', 'deleted', 'imgDeleted']) {
		delete post[key];
	}

	for (let key of ['num', 'time']) {
		post[key] = parseInt(post[key], 10);
	}
	imager.nestImageProps(post);
}

function postKey(num, op) {
	return `${op == num ? 'thread' : 'post'}:${num}`;
}

function destringifyList(list) {
	const parsed = [];
	for (let i = 0; i < list.length; i++) {
		// Stop gap fix until database migration
		try {
			parsed[i] = JSON.parse(list[i]);
		}
		catch (err) {
			winston.error(`Failed to destringify list: ${list.toString()}:`,
				err)
			return []
		}
	}
	return parsed;
}
exports.destrigifyList = destringifyList;

function forEachInObject(obj, f, callback) {
	var total = 0, complete = 0, done = false, errors = [];
	function cb(err) {
		complete++;
		if (err)
			errors.push(err);
		if (done && complete == total)
			callback(errors.length ? errors : null);
	}
	for (let k in obj) {
		if (obj.hasOwnProperty(k)) {
			total++;
			f(k, cb);
		}
	}
	done = true;
	if (complete == total)
		callback(errors.length ? errors : null);
}
