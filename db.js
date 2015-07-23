/*
Main redis controller module
 */

'use strict';

let _ = require('underscore'),
    async = require('async'),
    cache = require('./server/state').dbCache,
    caps = require('./server/caps'),
    common = require('./common'),
    config = require('./config'),
    events = require('events'),
    fs = require('fs'),
    hooks = require('./util/hooks'),
    hot = require('./server/state').hot,
    imager = require('./imager'),
    Muggle = require('./util/etc').Muggle,
    tail = require('./util/tail'),
    winston = require('winston');

let OPs = exports.OPs = cache.OPs,
	TAGS = exports.TAGS = cache.opTags,
	SUBS = exports.SUBS = cache.threadSubs;

function redis_client() {
	return require('redis').createClient(config.REDIS_PORT || undefined);
}
exports.redis_client = redis_client;
global.redis = redis_client();

exports.UPKEEP_IDENT = {
	auth: 'Upkeep',
	ip: '127.0.0.1'
};

// Depend on global redis client
let admin = require('./admin'),
	amusement = require('./server/amusement');

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
		this.subscription_callbacks.forEach(function(sub) {
			sub(null);
		});
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
		let k = this.k;
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
		this.subscription_callbacks.forEach(function(sub) {
			sub(err);
		});
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
			suffixPos = prefixLen + bodyLen;
		let info = {body: msg.substr(prefixLen, bodyLen)};
		if (msg.length > suffixPos)
			info.suffixPos = suffixPos;
		return info;
	}
	inject_extra(kind, msg, extra) {
		// Just one kind of insertion right now
		if (kind !== common.INSERT_POST || extra.ip)
			return null;

		// XXX: Why the fuck don't you just stringify arrays?
		let parsed = JSON.parse(`[${msg}]`);
		parsed[2].mnemonic = extra.mnemonic;
		return JSON.stringify(parsed).slice(1, -1);
	}
	has_no_listeners() {
		/* Possibly idle out after a while */
		let self = this;
		if (this.idleOutTimer)
			clearTimeout(this.idleOutTimer);
		this.idleOutTimer = setTimeout(function() {
			self.idleOutTimer = null;
			if (self.listeners('update').length == 0)
				self.commit_sudoku();
		}, 30000);
	}
	static get(target, ident) {
		const full = Subscription.full_key(target, ident);
		let sub = SUBS[full.key];
		if (!sub)
			sub = new Subscription(full);
		return sub;
	}
	static full_key(target, ident) {
		let channel;
		if (caps.checkAuth('janitor', ident))
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

function add_OP_tag(tagIndex, op) {
	var tags = TAGS[op];
	if (tags === undefined)
		TAGS[op] = tagIndex;
	else if (typeof tags == 'number') {
		if (tagIndex != tags)
			TAGS[op] = [tags, tagIndex];
	}
	else if (tags.indexOf(tagIndex) < 0)
		tags.push(tagIndex);
}

function set_OP_tag(tagIndex, op) {
	TAGS[op] = tagIndex;
}

function removeOPTag(op) {
	delete OPs[op];
	delete TAGS[op];
}

function OP_has_tag(tag, op) {
	var index = config.BOARDS.indexOf(tag);
	if (index < 0)
		return false;
	var tags = TAGS[op];
	if (tags === undefined)
		return false;
	if (typeof tags == 'number')
		return index == tags;
	else
		return tags.indexOf(index) >= 0;
}
exports.OP_has_tag = OP_has_tag;

function first_tag_of (op) {
	var tags = TAGS[op];
	if (tags === undefined)
		return false;
	else if (typeof tags == 'number')
		return config.BOARDS[tags];
	else
		return config.BOARDS[tags[0]];
}
exports.first_tag_of = first_tag_of;

function tags_of(op) {
	var tags = TAGS[op];
	if (tags === undefined)
		return false;
	else if (typeof tags == 'number')
		return [config.BOARDS[tags]];
	else
		return tags.map(function (i) { return config.BOARDS[i]; });
}
exports.tags_of = tags_of;


function track_OPs (callback) {
	var k = redis_client();
	k.subscribe('cache');
	k.once('subscribe', function () {
		load_OPs(callback);
	});
	k.on('message', update_cache);
	/* k persists for the purpose of cache updates */
}
exports.track_OPs = track_OPs;

function update_cache(chan, msg) {
	msg = JSON.parse(msg);
	var op = msg.op,
		kind = msg.kind,
		tag = config.BOARDS.indexOf(msg.tag);

	if (kind == common.INSERT_POST) {
		if (msg.num)
			OPs[msg.num] = op;
		else {
			add_OP_tag(tag, op);
			OPs[op] = op;
		}
	}
	else if (kind == common.DELETE_POSTS) {
		const nums = msg.nums;
		for (let i = 0, l = msg.num.length; i < l; i++) {
			delete OPs[nums[i]];
		}
	}
	else if (kind == common.DELETE_THREAD) {
		const nums = msg.nums;
		for (let i = 0, l = nums.length; i < l; i++) {
			delete OPs[nums[i]];
		}
		delete TAGS[op];
	}
}

function on_pub (name, handler) {
	// TODO: share redis connection
	var k = redis_client();
	k.subscribe(name);
	k.on('message', handler);
	/* k persists */
}
exports.on_pub = on_pub;

function load_OPs(callback) {
	var r = global.redis;
	var boards = config.BOARDS;
	// Want consistent ordering in the TAGS entries for multi-tag threads
	// (so do them in series)
	tail.forEach(boards, scan_board, callback);

	var threadsKey;
	function scan_board(tag, cb) {
		var tagIndex = boards.indexOf(tag);
		threadsKey = 'tag:' + tag_key(tag) + ':threads';
		r.zrange(threadsKey, 0, -1, function (err, threads) {
			if (err)
				return cb(err);
			async.forEach(threads, function (op, cb) {
				op = parseInt(op, 10);
				var ps = [scan_thread.bind(null,tagIndex,op)];
				async.parallel(ps, cb);
			}, cb);
		});
	}

	function scan_thread(tagIndex, op, cb) {
		op = parseInt(op, 10);
		add_OP_tag(tagIndex, op);
		OPs[op] = op;
		get_all_replies(r, op, function (err, posts) {
			if (err)
				return cb(err);
			for (let i = 0, l = posts.length; i < l; i++) {
				OPs[parseInt(posts[i], 10)] = op;
			}
			cb(null);
		});
	}
}

/* SOCIETY */

class Yakusoku extends events.EventEmitter {
	constructor(board, ident) {
		super();
		this.id = ++(cache.YAKUMAN);
		this.tag = board;
		this.ident = ident;
		this.subs = [];
	}
	connect() {
		// multiple redis connections are pointless (without slaves)
		return global.redis;
	}
	disconnect() {
		this.removeAllListeners();
	}
	kiku(targets, on_update, on_sink, callback) {
		let self = this;
		this.on_update = on_update;
		this.on_sink = on_sink;
		forEachInObject(targets, function(id, cb) {
			const target = self.target_key(id);
			let sub = Subscription.get(target, self.ident);
			sub.on('update', on_update);
			sub.on('error', on_sink);
			self.subs.push(sub.fullKey);
			sub.when_ready(cb);
		}, callback);
	}
	target_key(id) {
		return id === 'live' ? 'tag:' + this.tag : 'thread:' + id;
	}
	kikanai() {
		const subs = this.subs;
		for (let i = 0, l = subs.length; i < l; i++) {
			let sub = SUBS[subs[i]];
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
		let r = this.connect();
		if (ip === '127.0.0.1')
			return reserve();

		const key = `ip:${ip}:throttle:`,
			now = Date.now(),
			shortTerm = key + this.short_term_timeslot(now),
			longTerm = key + this.long_term_timeslot(now);
		r.mget([shortTerm, longTerm], function(err, quants) {
			if (err)
				return callback(Muggle("Limiter failure.", err));
			if (quants[0] > config.SHORT_TERM_LIMIT
				|| quants[1] > config.LONG_TERM_LIMIT
			)
				return callback(Muggle('Reduce your speed.'));

			reserve();
		});

		function reserve() {
			r.incr('postctr', function(err, num) {
				if (err)
					return callback(err);
				OPs[num] = op || num;
				callback(null, num);
			});
		}
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
		let r = this.connect();
		if (!this.tag)
			return callback(Muggle("Can't retrieve board for posting."));
		let op = msg.op;
		const ip = extra.ip,
			board = extra.board,
			num = msg.num,
			isThead = !op;
		if (!op)
			op = num;
		if (!num)
			return callback(Muggle("No post number."));
		else if (!ip)
			return callback(Muggle("No IP."));
		else if (!isThead && (OPs[op] != op || !OP_has_tag(board, op))) {
			delete OPs[num];
			return callback(Muggle('Thread does not exist.'));
		}

		let view = {
			time: msg.time,
			num: num,
			board: board,
			ip: ip,
			state: msg.state.join()
		};
		const optPostFields = [
			'name', 'trip', 'email', 'auth', 'subject', 'dice'
		];
		for (let i = 0, l = optPostFields.length; i < l; i++) {
			const field = optPostFields[i];
			if (msg[field])
				view[field] = msg[field];
		}
		const tagKey = 'tag:' + tag_key(this.tag);
		if (isThead)
			view.tags = tag_key(board);
		else
			view.op = op;

		if (extra.image_alloc) {
			msg.image = extra.image_alloc.image;
			if (isThead == msg.image.pinky)
				return callback(Muggle("Image is the wrong size."));
			delete msg.image.pinky;
		}

		const key = (isThead ? 'thread:' : 'post:') + num;
		let m = r.multi();
		m.incr(tagKey + ':postctr'); // must be first
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

		let etc = {
			augments: {},
			cacheUpdate: {}
		};
		if (isThead) {
			// TODO: Add to alternate thread list?

			/* Rate-limit new threads */
			if (ip != '127.0.0.1')
				m.setex('ip:'+ip+':throttle:thread', config.THREAD_THROTTLE, op);
		}
		else {
			etc.cacheUpdate.num = num;
			m.rpush(`thread:${op}:posts`, num);
		}

		/* Denormalize for backlog */
		view.nonce = msg.nonce;
		view.body = body;

		let self = this,
			bump;
		async.waterfall(
			[
				function (next) {
					if (!msg.image)
						return next(null);
					imager.commit_image_alloc(extra.image_alloc, next);
				},
				// Determine, if we need to bump the thread to the top of
				// the board
				function(next) {
					if (isThead) {
						bump = true;
						return next();
					}

					r.llen(`thread:${op}:posts`, function(err, res) {
						if (err)
							return next(err);
						bump = !common.is_sage(view.email)
							&& res < config.BUMP_LIMIT[board];
						next();
					});
				},
				function(next) {
					if (ip) {
						const n = self.post_volume(view, body);
						if (n > 0)
							self.update_throughput(m, ip, view.time, n);

						// Only the client-private Reader() instances need
						// to embed mnemonics in-post. Doing that here would
						// publish it to everyone. Instead live mnemonic
						// updates are pushed through the 'auth' channel to
						// authenticated staff only.
						const mnemonic = admin.genMnemonic(ip);
						if (mnemonic)
							etc.augments.auth = {mnemonic};
					}

					// Don't parse dice, because they aren't stringified on
					// live publishes
					extract(view, true);
					if (bump)
						m.incr(tagKey + ':bumpctr');
					self._log(m, op, common.INSERT_POST, [view, bump], etc);
					m.exec(next);
				},
				function(res, next) {
					if (!bump)
						return next();
					r.zadd(tagKey + ':threads', res[0], op, next);
				}
			],
			function (err) {
				if (err) {
					delete OPs[num];
					return callback(err);
				}
				callback(null);
			}
		);
	}
	imageDuplicateHash(m, hash, num) {
		m.zadd('imageDups',
			Date.now() + (config.DEBUG ? 30000 : 3600000),
			num + ':' + hash
		);
	}
	writeDice(m, dice, key) {
		let stringified = [];
		for (let i = 0, l = dice.length; i < l; i++) {
			stringified[i] = JSON.stringify(dice[i]);
		}
		m.lpush(key + ':dice', stringified);
	}
	addBacklinks(m, num, op, links) {
		for (let targetNum in links) {
			// Check if post exists through cache
			if (!(targetNum in OPs))
				continue;
			const key = (targetNum in TAGS ? 'thread' : 'post')
				+ `:${targetNum}:backlinks`;
			m.hset(key, num, op);
			this._log(m, links[targetNum], common.BACKLINK, [targetNum, num, op]);
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
	_log(m, op, kind, msg, opts) {
		opts = opts || {};
		msg = JSON.stringify(msg).slice(1, -1);
		msg = msg.length ? (kind + ',' + msg) : ('' + kind);
		if (config.DEBUG)
			winston.info("Log: " + msg);
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
		const tags = opts.tags || (this.tag ? [this.tag] : []);
		for (let i = 0, l = tags.length; i < l; i++) {
			m.publish('tag:' + tags[i], msg);
		}
		if (opts.cacheUpdate) {
			var info = {kind: kind, tag: tags[0], op: op};
			_.extend(info, opts.cacheUpdate);
			m.publish('cache', JSON.stringify(info));
		}
	}
	add_image(post, alloc, ip, callback) {
		let r = this.connect();
		const num = post.num,
			op = post.op;
		if (!op)
			return callback(Muggle("Can't add another image to an OP."));
		let image = alloc.image;
		if (!image.pinky)
			return callback(Muggle("Image is wrong size."));
		delete image.pinky;

		const key = 'post:' + num;
		let self = this;
		async.waterfall([
			function(next) {
				r.exists(key, next);
			},
			function(exists, next) {
				if (!exists)
					return next(Muggle("Post does not exist."));
				imager.commit_image_alloc(alloc, next);
			},
			function(next) {
				let m = r.multi();
				self.imageDuplicateHash(m, image.hash, num);
				m.hmset(key, image);
				m.hincrby('thread:' + op, 'imgctr', 1);

				// Useless on the client
				delete image.hash;
				self._log(m, op, common.INSERT_IMAGE, [num, image]);

				const now = Date.now();
				self.update_throughput(m, ip, now,
					self.post_volume({image: true})
				);
				m.exec(next);
			}
		], callback);
	}
	append_post(post, tail, old_state, extra, cb) {
		let m = this.connect().multi();
		const key = (post.op ? 'post:' : 'thread:') + post.num;
		/* Don't need to check .exists() thanks to client state */
		m.append(key + ':body', tail);
		/* XXX: fragile */
		if (old_state[0] != post.state[0] || old_state[1] != post.state[1])
			m.hset(key, 'state', post.state.join());
		if (extra.ip) {
			const now = Date.now();
			this.update_throughput(m, extra.ip, now,
				this.post_volume(null, tail)
			);
		}
		if (!_.isEmpty(extra.new_links))
			m.hmset(key + ':links', extra.new_links);

		const num = post.num,
			op = post.op || num;
		// TODO: Make less dirty, when post state is refactored
		let _extra = {
			state: [old_state[0] || 0, old_state[1] || 0]
		};
		const links = extra.links;
		if (links) {
			_extra.links = links;
			this.addBacklinks(m, num, op, links);
		}
		const dice = extra.dice;
		if (dice) {
			_extra.dice = dice;
			this.writeDice(m, dice, key);
		}

		this._log(m, op, common.UPDATE_POST, [num, tail, _extra]);
		m.exec(cb);
	}
	finish_post(post, callback) {
		let m = this.connect().multi();
		const key = (post.op ? 'post:' : 'thread:') + post.num;
		/* Don't need to check .exists() thanks to client state */
		this.finish_off(m, key, post.body);
		this._log(m, post.op || post.num, common.FINISH_POST, [post.num]);
		m.exec(callback);
	}
	finish_off(m, key, body) {
		m.hset(key, 'body', body);
		m.del(key.replace('dead', 'thread') + ':body');
		m.hdel(key, 'state');
		m.srem('liveposts', key);
	}
	finish_quietly(key, callback) {
		let r = this.connect(),
			self = this;
		async.waterfall([
			function(next) {
				r.hexists(key, 'body', next);
			},
			function(exists, next) {
				if (exists)
					return callback(null);
				r.get(key.replace('dead', 'thread') + ':body', next);
			},
			function(body, next) {
				let m = r.multi();
				self.finish_off(m, key, body);
				m.exec(next);
			}
		], callback);
	}
	finish_all(callback) {
		let r = this.connect(),
			self = this;
		r.smembers('liveposts', function(err, keys) {
			if (err)
				return callback(err);
			async.forEach(keys, function(key, cb) {
				let m = r.multi();
				m.get(key + ':body');
				const isPost = key.slice(0, 5) == 'post:';
				if (isPost)
					m.hget(key, 'op');
				m.exec(function(err, rs) {
					if (err)
						return cb(err);
					let m = r.multi();
					self.finish_off(m, key, rs[0]);
					const n = parseInt(key.match(/:(\d+)$/)[1]),
						op = isPost ? parseInt(rs[1], 10) : n;
					self._log(m, op, common.FINISH_POST, [n]);
					m.srem('liveposts', key);
					m.exec(cb);
				});
			}, callback);
		});
	}
	fetch_backlogs(watching, callback) {
		let r = this.connect(),
			combined = [];
		forEachInObject(watching,
			function (thread, cb) {
				if (thread == 'live')
					return cb(null);
				const key = 'thread:' + thread + ':history',
					sync = watching[thread];
				r.lrange(key, sync, -1, function (err, log) {
					if (err)
						return cb(err);
					const prefix = thread + ',';
					for (let i = 0, l = log.length; i < l; i++) {
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
		this.connect().hexists('thread:' + op, 'locked', function(err, lock) {
			if (err)
				return callback(err);
			callback(lock ? Muggle('Thread is locked.') : null);
		});
	}
	check_throttle(ip, callback) {
		// So we can spam new threads in debug mode
		if (config.DEBUG)
			return callback(null);
		const key = `ip:${ip}:throttle:thread`;
		this.connect().exists(key, function(err, exists) {
			if (err)
				return callback(err);
			callback(exists ? Muggle('Too soon.') : null);
		});
	}
	get_tag(page) {
		let r = this.connect(),
			self = this;
		const keyBase = 'tag:' + tag_key(this.tag),
			key = keyBase + ':threads';

		// -1 is for live pages and -2 is for catalog
		const catalog = page === -2;
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

		let m = r.multi();
		m.zrevrange(key, start, end);
		m.zcard(key);
		// Used for building board eTags
		m.get(keyBase + ':postctr');
		m.exec(function (err, res) {
			if (err)
				return self.emit('error', err);
			let nums = res[0];
			if (page > 0 && !nums.length)
				return self.emit('nomatch');
			self.emit('begin', res[1] || 0, res[2] || 0);
			let reader = new Reader(self.ident);
			reader.on('error', self.emit.bind(self, 'error'));
			reader.on('thread', self.emit.bind(self, 'thread'));
			reader.on('post', self.emit.bind(self, 'post'));
			reader.on('endthread', self.emit.bind(self, 'endthread'));
			self._get_each_thread(reader, 0, nums, catalog);
		});
	}
	_get_each_thread(reader, ix, nums, catalog) {
		if (!nums || ix >= nums.length) {
			this.emit('end');
			reader.removeAllListeners('endthread');
			reader.removeAllListeners('end');
			return;
		}

		let self = this;
		function next_please() {
			reader.removeListener('end', next_please);
			reader.removeListener('nomatch', next_please);
			self._get_each_thread(reader, ix+1, nums, catalog);
		}

		reader.on('end', next_please);
		reader.on('nomatch', next_please);
		reader.get_thread(this.tag, nums[ix], {
			catalog,
			abbrev: hot.ABBREVIATED_REPLIES || 5
		});
	}
	// Purges all the thread's keys from the database and delete's all images
	// contained
	purge_thread(op, board, callback) {
		let r = this.connect();
		const key = 'thread:' + op;
		let keysToDel = [],
			filesToDel = [],
			nums = [];
		async.waterfall([
			// Confirm thread can be deleted
			function(next) {
				r.exists(key, next);
			},
			function(res, next) {
				// Likely to happen, if interrupted mid-purge
				if (!res) {
					r.zrem(`tag:${tag_key(board)}:threads`, op);
					return callback();
				}
				// Get reply list
				r.lrange(key + ':posts', 0, -1, next);
			},
			// Read all post hashes
			function(posts, next) {
				let m = r.multi();
				for (let i = 0, l = posts.length; i < l; i++) {
					// Queue for removal from post cache
					nums.push(posts[i]);
					posts[i] = 'post:' + posts[i];
				}
				// Parse OP key like all other hashes. `res` will always be an
				// array, even if empty.
				posts.unshift(key);
				for (let i = 0, l = posts.length; i < l; i++) {
					const key = posts[i];
					m.hgetall(key);
					m.exists(key + ':links');
					m.exists(key + ':backlinks');
					// It should only still exists because of server shutdown
					// mid-post, but those do happen
					m.exists(key + ':body');
					m.exists(key + ':dice');
				}
				// A bit more complicated, because we need to pass two arguments
				// to the next function, to map the arrays
				m.exec(function(err, res) {
					if (err)
						return next(err);
					next(null, res, posts);
				})
			},
			// Populate key and file to delete arrays
			function(res, posts, next) {
				const imageTypes = ['src', 'thumb', 'mid'],
					optional = [':links', ':backlinks', ':body', ':dice'];
				let path = imager.media_path;
				for (let i = 0, l = res.length; i < l; i += 6) {
					const hash = res[i],
						key = posts[i / 6];
					if (!hash)
						continue;

					keysToDel.push(key);
					for (let o = 0; o < optional.length; o++) {
						if (!res[i + o])
							continue;
						keysToDel.push(key + optional[o]);
					}

					// Add images to delete list
					for (let o = 0, len = imageTypes.length; o < len; o++) {
						const type = imageTypes[o],
							image = hash[type];
						if (!image)
							continue;
						filesToDel.push(path(type, image));
					}
				}
				next();
			},
			// Check for OP-only keys
			function(next) {
				const suffixes = ['dels', 'history', 'posts'];
				let OPKeys = [],
					m = r.multi();
				for (let i = 0, l = suffixes.length; i < l; i++) {
					OPKeys.push(`${key}:${suffixes[i]}`);
				}
				for (let i = 0, l = OPKeys.length; i < l; i++) {
					m.exists(OPKeys[i]);
				}
				m.exec(function(err, res) {
					if (err)
						return next(err);
					next(null, res, OPKeys);
				})
			},
			function(res, OPKeys, next) {
				let keys = keysToDel;
				for (let i = 0, l = res.length; i < l; i++) {
					if (res[i])
						keys.push(OPKeys[i]);
				}

				// Delete all keys
				let m = r.multi();
				for (let i = 0, l = keys.length; i < l; i++) {
					m.del(keys[i]);
				}
				m.exec(next);
			},
			function(res, next) {
				// Delete all images
				async.each(filesToDel,
					function(file, cb) {
						fs.unlink(file, function(err) {
							if (err)
								return cb(err);
							cb();
						});
					},
					function(err) {
						if (err)
							return next(err);
						next();
					}
				);
			},
			function(next) {
				// Delete thread entry from the set
				r.zrem(`tag:${tag_key(board)}:threads`, op, next);
			},
			function(res, next) {
				// Clear thread and post numbers from caches
				for (let i = 0, l = nums.length; i < l; i++) {
					delete OPs[nums];
				}
				removeOPTag(op);
				next();
			}
		], callback);
	}
	get_fun(op, callback) {
		if (cache.funThread && op == cache.funThread) {
			/* Don't cache, for extra fun */
			fs.readFile('client/fun.js', 'UTF-8', callback);
		}
		else
			callback(null);
	}
	get_banner(cb) {
		this.connect().get('banner:info', cb);
	}
	set_banner(message, cb) {
		let r = this.connect(),
			self = this;
		r.set('banner:info', message, function(err) {
			if (err)
				return cb(err);
			// Dispatch new banner
			let m = r.multi();
			self._log(m, 0, common.UPDATE_BANNER, [message]);
			m.exec(cb);
		});
	}
	modHandler(method, nums, cb) {
		// Group posts by thread for live publishes to the clients
		let threads = {};
		for (let num of nums) {
			const op = OPs[num];
			if (!(op in threads))
				threads[op] = [];
			threads[op].push(num);
		}
		async.forEachOf(threads, this[method].bind(this), cb);
		return true;
	}
	spoilerImages(nums, op, cb) {
		let r = this.connect(),
			m = r.multi(),
			keys = [];
		for (let num of nums) {
			const key = postKey(num, op);
			keys.push(key);
			m.hmget(key, 'src', 'spoiler');
		}
		let self = this;
		m.exec(function (err, data) {
			if (err)
				return cb(err);
			let m = r.multi(),
				updates = [];
			for (let i = 0; i < data.length; i++) {
				// No image or already spoilt
				if (!data[i][0] || data[i][1])
					continue;
				const spoiler = common.pick_spoiler(-1).index;
				m.hset(keys[i], 'spoiler', spoiler);
				updates.push(nums[i], spoiler);
			}
			if (updates.length)
				self._log(m, op, common.SPOILER_IMAGES, updates);
			m.exec(cb);
		});
	}
	deleteImages(nums, op, cb) {
		let r = this.connect(),
			m = r.multi(),
			keys = [],
			self = this;
		for (let num of nums) {
			const key = postKey(num, op);
			keys.push(key);
			m.hmget(key, 'src', 'imgDeleted');
		}
		m.exec(function (err, data) {
			if (err)
				return cb(err);
			let updates = [];
			for (let i = 0; i < data.length; i++) {
				// No image or already hidden
				if (!data[i][0] || data[i][1])
					continue;
				m.hset(keys[i], 'imgDeleted', true);
				updates.push(nums[i]);
			}
			if (updates.length)
				self._log(m, op, common.DELETE_IMAGES, updates);
			m.exec(cb);
		});
	}
}
exports.Yakusoku = Yakusoku;

/* LURKERS */

class Reader extends events.EventEmitter {
	constructor(ident) {
		// Call the EventEmitter's constructor
		super();
		this.canModerate = caps.checkAuth('janitor', ident);
		this.r = global.redis;
	}
	get_thread(tag, num, opts) {
		let r = this.r;
		const graveyard = tag === 'graveyard';
		if (graveyard)
			opts.showDead = true;
		const key = (graveyard ? 'dead:' : 'thread:') + num;
		let self = this;
		r.hgetall(key, function(err, pre_post) {
			if (err)
				return self.emit('error', err);
			if (_.isEmpty(pre_post)) {
				if (!opts.redirect)
					return self.emit('nomatch');
				r.hget('post:' + num, 'op', function(err, op) {
					if (err)
						self.emit('error', err);
					else if (!op)
						self.emit('nomatch');
					else
						self.emit('redirect', op);
				});
				return;
			}
			let exists = true;
			if (pre_post.hide && !opts.showDead)
				exists = false;
			const tags = parse_tags(pre_post.tags);
			if (!graveyard && tags.indexOf(tag) < 0) {
				/* XXX: Should redirect directly to correct thread */
				if (opts.redirect)
					return self.emit('redirect', num, tags[0]);
				else
					exists = false;
			}
			if (!exists) {
				self.emit('nomatch');
				return;
			}
			self.emit('begin', pre_post);

			pre_post.time = parseInt(pre_post.time, 10);

			let nums, deadNums, opPost,
				total = 0;
			const abbrev = opts.abbrev || 0;
			async.waterfall(
				[
					function (next) {
						self.with_body(key, pre_post, next);
					},
					function (fullPost, next) {
						opPost = fullPost;
						let m = r.multi();
						const postsKey = key + ':posts';

						// order is important!
						m.lrange(postsKey, -abbrev, -1);
						// The length of the above array is limited by the
						// amount of posts we are retrieving. A total number
						// of posts is quite useful.
						m.llen(postsKey);
						self.getExtras(m, key);
						if (abbrev)
							m.llen(postsKey);
						if (opts.showDead) {
							var deadKey = key + ':dels';
							m.lrange(deadKey, -abbrev, -1);
							if (abbrev)
								m.llen(deadKey);
						}
						m.exec(next);
					},
					function (rs, next) {
						// get results in the same order as before
						nums = rs.shift();
						// NOTE: these are only the displayed replies, not
						// all of them
						opPost.replies = nums || [];
						opPost.replyctr = parseInt(rs.shift(), 10) || 0;
						self.parseExtras(rs, opPost);
						if (abbrev)
							total += parseInt(rs.shift(), 10);
						if (opts.showDead) {
							deadNums = rs.shift();
							if (abbrev)
								total += parseInt(rs.shift(), 10);
						}

						self.injectMnemonic(opPost);
						extract(opPost);
						opPost.omit = Math.max(total - abbrev, 0);
						opPost.hctr = parseInt(opPost.hctr, 10);
						// So we can pass a thread number on `endthread`
						// emission
						opts.op = opPost.num;
						next(null);
					}
				],
				function (err) {
					if (err)
						return self.emit('error', err);
					self.emit('thread', opPost);
					if (opts.catalog)
						return self.emit('end');
					if (deadNums)
						nums = self.merge_posts(nums, deadNums, abbrev);
					self._get_each_reply(tag, 0, nums, opts);
				}
			);
		});
	}
	getExtras(m, key) {
		m.hgetall(key + ':links');
		m.hgetall(key + ':backlinks');
		m.lrange(key + ':dice', 0, -1);
	}
	parseExtras(res, post) {
		for (let key of ['links', 'backlinks', 'dice']) {
			const prop = res.shift();
			if (prop)
				post[key] = prop;
		}
	}
	injectMnemonic(post) {
		if (!this.canModerate)
			return;
		const mnemonic = admin.genMnemonic(post.ip);
		if (mnemonic)
			post.mnemonic = mnemonic;
	}
	merge_posts(nums, deadNums, abbrev) {
		let i = nums.length - 1,
			pi = deadNums.length - 1;
		if (pi < 0)
			return nums;
		let merged = [];
		while (!abbrev || merged.length < abbrev) {
			if (i >= 0 && pi >= 0) {
				const num = nums[i],
					pNum = deadNums[pi];
				if (parseInt(num, 10) > parseInt(pNum, 10)) {
					merged.unshift(num);
					i--;
				}
				else {
					merged.unshift(pNum);
					pi--;
				}
			}
			else if (i >= 0)
				merged.unshift(nums[i--]);
			else if (pi >= 0)
				merged.unshift(deadNums[pi--]);
			else
				break;
		}
		return merged;
	}
	_get_each_reply(tag, ix, nums, opts) {
		if (!nums || ix >= nums.length) {
			this.emit('endthread', opts.op);
			this.emit('end');
			return;
		}
		const num = parseInt(nums[ix], 10);
		let self = this;
		this.get_post('post', num, opts, function (err, post) {
			if (err)
				return self.emit('error', err);
			if (post)
				self.emit('post', post);
			self._get_each_reply(tag, ix + 1, nums, opts);
		});
	}
	get_post(kind, num, opts, cb) {
		let r = this.r,
			self = this;
		const key = kind + ':' + num;
		async.waterfall([
			function (next) {
				let m = r.multi();
				m.hgetall(key);
				self.getExtras(m, key);
				m.exec(next);
			},
			function (data, next) {
				let pre_post = data.shift();
				self.parseExtras(data, pre_post);
				let exists = !(_.isEmpty(pre_post));
				if (exists && pre_post.hide && !opts.showDead)
					exists = false;
				if (!exists)
					return next(null, null);

				pre_post.num = num;
				pre_post.time = parseInt(pre_post.time, 10);
				if (kind === 'post')
					pre_post.op = parseInt(pre_post.op, 10);
				else {
					/*
					 TODO: filter by ident eligibility and attach
					 Currently used only for reporting
					 */
					//var tags = parse_tags(pre_post.tags);
				}
				self.with_body(key, pre_post, next);
			},
			function (post, next) {
				if (post) {
					self.injectMnemonic(post);

					// Image is deleted and client not authenticated
					if (post.imgDeleted && !self.canModerate)
						imager.deleteImageProps(post);
					extract(post);
				}
				next(null, post);
			}
		],	cb);
	}
	with_body(key, post, callback) {
		if (post.body !== undefined)
			return callback(null, post);

		let r = this.r;
		r.get(key + ':body', function(err, body) {
			if (err)
				return callback(err);
			if (body !== null) {
				post.body = body;
				post.editing = true;
				return callback(null, post);
			}
			// Race condition between finishing posts
			r.hget(key, 'body', function(err, body) {
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
			return cb(null);
		this.get_post(key, num, {}, function(err, post) {
			if (err || !post)
				return cb(null);
			cb(post);
		})
	}
}
exports.Reader = Reader;

// Retrieve post info from cache
function postInfo(num) {
	const isOP = num in TAGS;
	return {
		isOP,
		board: config.BOARDS[isOP ? TAGS[num] : TAGS[OPs[num]]]
	};
}
exports.postInfo = postInfo;

/* HELPERS */

function is_board (board) {
	return config.BOARDS.indexOf(board) >= 0;
}
exports.is_board = is_board;

function get_all_replies(r, op, cb) {
	var key = 'thread:' + op;
	r.lrange(key + ':posts', 0, -1, function(err, nums) {
		if (err)
			return cb(err);
		return cb(null, nums);
	});
}

function extract(post, dontParseDice) {
	delete post.ip;
	post.num = parseInt(post.num, 10);
	imager.nestImageProps(post);
	if (!dontParseDice)
		amusement.parseDice(post);
}

function postKey(num, op) {
	return `${op == num ? 'thread' : 'post'}:${num}`;
}

function tag_key(tag) {
	return tag.length + ':' + tag;
}
exports.tag_key = tag_key;

function parse_tags(input) {
	if (!input) {
		winston.warn('Blank tag!');
		return [];
	}
	var tags = [];
	while (input.length) {
		var m = input.match(/^(\d+):/);
		if (!m)
			break;
		var len = parseInt(m[1], 10);
		var pre = m[1].length + 1;
		if (input.length < pre + len)
			break;
		tags.push(input.substr(pre, len));
		input = input.slice(pre + len);
	}
	return tags;
}
exports.parse_tags = parse_tags;

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
