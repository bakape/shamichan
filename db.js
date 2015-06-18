'use strict';

var _ = require('underscore'),
    async = require('async'),
    cache = require('./server/state').dbCache,
    caps = require('./server/caps'),
    common = require('./common'),
    config = require('./config'),
    events = require('events'),
    fs = require('fs'),
    hooks = require('./util/hooks'),
    hot = require('./server/state').hot,
	// set up hooks
    imager = require('./imager'),
    Muggle = require('./util/etc').Muggle,
    tail = require('./util/tail'),
    util = require('util'),
    winston = require('winston');

var OPs = exports.OPs = cache.OPs;
var TAGS = exports.TAGS = cache.opTags;
var SUBS = exports.SUBS = cache.threadSubs;

function redis_client() {
	return require('redis').createClient(config.REDIS_PORT || undefined);
}
exports.redis_client = redis_client;

global.redis = redis_client();

/* REAL-TIME UPDATES */

function Subscription(targetInfo) {
	events.EventEmitter.call(this);
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
	this.subscriptions = [this.target];
	this.pending_subscriptions.push(this.target);
	if (this.target != this.fullKey) {
		this.k.subscribe(this.fullKey);
		this.pending_subscriptions.push(this.fullKey);
	}
}

util.inherits(Subscription, events.EventEmitter);
var S = Subscription.prototype;

Subscription.full_key = function (target, ident) {
	var channel;
	if (caps.can_moderate(ident))
		channel = 'auth';
	var key = channel ? channel + ':' + target : target;
	return {key: key, channel: channel, target: target};
};

Subscription.get = function (target, ident) {
	var full = Subscription.full_key(target, ident);
	var sub = SUBS[full.key];
	if (!sub)
		sub = new Subscription(full);
	return sub;
};

S.when_ready = function (cb) {
	if (this.subscription_callbacks)
		this.subscription_callbacks.push(cb);
	else
		cb(null);
};

S.on_one_sub = function (name) {
	var i = this.pending_subscriptions.indexOf(name);
	if (i < 0)
		throw "Obtained unasked-for subscription " + name + "?!";
	this.pending_subscriptions.splice(i, 1);
	if (this.pending_subscriptions.length == 0)
		this.on_all_subs();
};

S.on_all_subs = function () {
	let k = this.k;
	k.removeAllListeners('subscribe');
	k.on('message', this.on_message.bind(this));
	k.removeAllListeners('error');
	k.on('error', this.sink_sub.bind(this));
	for (let i = 0, subCs = this.subscription_callbacks, l = subCs.length;
		  i < l; i++) {
		subCs[i](null);
	}
	delete this.pending_subscriptions;
	delete this.subscription_callbacks;
};

function parse_pub_message(msg) {
	var m = msg.match(/^(\d+)\|/);
	var prefixLen = m[0].length;
	var bodyLen = parseInt(m[1], 10);
	var info = {body: msg.substr(prefixLen, bodyLen)};
	var suffixPos = prefixLen + bodyLen;
	if (msg.length > suffixPos)
		info.suffixPos = suffixPos;
	return info;
}

S.on_message = function (chan, msg) {
	/* Do we need to clarify whether this came from target or fullKey? */
	var parsed = parse_pub_message(msg), extra;
	if (this.channel && parsed.suffixPos) {
		var suffix = JSON.parse(msg.slice(parsed.suffixPos));
		extra = suffix[this.channel];
	}
	msg = parsed.body;
	var m = msg.match(/^(\d+),(\d+)/);
	var op = parseInt(m[1], 10);
	var kind = parseInt(m[2], 10);

	if (extra) {
		var modified = inject_extra(kind, msg, extra);
		// currently this won't modify op or kind,
		// but will have to watch out for that if that changes
		if (modified)
			msg = modified;
	}
	this.emit('update', op, kind, '[[' + msg + ']]');
};

function inject_extra(kind, msg, extra) {
	// Just one kind of insertion right now
	if (kind == common.INSERT_POST && extra.ip) {
		var m = msg.match(/^(\d+,\d+,\d+,)(.+)$/);
		var post = JSON.parse(m[2]);
		post.ip = extra.ip;
		return m[1] + JSON.stringify(post);
	}
}

S.on_sub_error = function (err) {
	winston.error("Subscription error:", (err.stack || err));
	this.commit_sudoku();
	for (let i = 0, subCs = this.subscription_callbacks, l = subCs.length;
		  i < l; i++) {
		subCs[i](err);
	}
	this.subscription_callbacks = null;
};

S.sink_sub = function (err) {
	if (config.DEBUG)
		throw err;
	this.emit('error', this.target, err);
	this.commit_sudoku();
};

S.commit_sudoku = function () {
	var k = this.k;
	k.removeAllListeners('error');
	k.removeAllListeners('message');
	k.removeAllListeners('subscribe');
	k.quit();
	if (SUBS[this.fullKey] === this)
		delete SUBS[this.fullKey];
	this.removeAllListeners('update');
	this.removeAllListeners('error');
};

S.has_no_listeners = function () {
	/* Possibly idle out after a while */
	var self = this;
	if (this.idleOutTimer)
		clearTimeout(this.idleOutTimer);
	this.idleOutTimer = setTimeout(function () {
		self.idleOutTimer = null;
		if (self.listeners('update').length == 0)
			self.commit_sudoku();
	}, 30 * 1000);
};

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


function is_board (board) {
	return config.BOARDS.indexOf(board) >= 0;
}
exports.is_board = is_board;

exports.UPKEEP_IDENT = {auth: 'Upkeep', ip: '127.0.0.1'};

function Yakusoku(board, ident) {
	events.EventEmitter.call(this);
	this.id = ++(cache.YAKUMAN);
	this.tag = board;
	this.ident = ident;
	this.subs = [];
}

util.inherits(Yakusoku, events.EventEmitter);
exports.Yakusoku = Yakusoku;
var Y = Yakusoku.prototype;

Y.connect = function () {
	// multiple redis connections are pointless (without slaves)
	return global.redis;
};

Y.disconnect = function () {
	this.removeAllListeners();
};

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

Y.target_key = function (id) {
	return (id == 'live') ? 'tag:' + this.tag : 'thread:' + id;
};

Y.kiku = function (targets, on_update, on_sink, callback) {
	var self = this;
	this.on_update = on_update;
	this.on_sink = on_sink;
	forEachInObject(targets, function (id, cb) {
		var target = self.target_key(id);
		var sub = Subscription.get(target, self.ident);
		sub.on('update', on_update);
		sub.on('error', on_sink);
		self.subs.push(sub.fullKey);
		sub.when_ready(cb);
	}, callback);
};

Y.kikanai = function () {
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
};

function post_volume(view, body) {
	return (body ? body.length : 0) +
		(view ? (config.NEW_POST_WORTH || 0) : 0) +
		((view && view.image) ? (config.IMAGE_WORTH || 0) : 0);
}

function update_throughput(m, ip, when, quant) {
	var key = 'ip:' + ip + ':throttle:';
	var shortKey = key + short_term_timeslot(when);
	var longKey = key + long_term_timeslot(when);
	m.incrby(shortKey, quant);
	m.incrby(longKey, quant);
	/* Don't want to use expireat in case of timezone trickery
	   or something dumb. (Really, UTC should be OK though...) */
	// Conservative expirations
	m.expire(shortKey, 10 * 60);
	m.expire(longKey, 2 * 24 * 3600);
}

function short_term_timeslot(when) {
	return Math.floor(when / (1000 * 60 * 5));
}

function long_term_timeslot(when) {
	return Math.floor(when / (1000 * 60 * 60 * 24));
}

Y.reserve_post = function (op, ip, callback) {
	if (config.READ_ONLY)
		return callback(Muggle("Can't post right now."));
	var r = this.connect();
	if (ip == '127.0.0.1')
		return reserve();

	var key = 'ip:' + ip + ':throttle:';
	var now = Date.now();
	var shortTerm = key + short_term_timeslot(now);
	var longTerm = key + long_term_timeslot(now);
	r.mget([shortTerm, longTerm], function (err, quants) {
		if (err)
			return callback(Muggle("Limiter failure.", err));
		if (quants[0] > config.SHORT_TERM_LIMIT ||
				quants[1] > config.LONG_TERM_LIMIT)
			return callback(Muggle('Reduce your speed.'));

		reserve();
	});

	function reserve() {
		r.incr('postctr', function (err, num) {
			if (err)
				return callback(err);
			OPs[num] = op || num;
			callback(null, num);
		});
	}
};

var optPostFields = 'name trip email auth subject'.split(' ');

Y.insert_post = function (msg, body, extra, callback) {
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
	for (let i = 0, l = optPostFields.length; i < l; i++) {
		const field = optPostFields[i];
		if (msg[field])
			view[field] = msg[field];
	}
	const tagKey = 'tag:' + tag_key(this.tag);
	if (isThead) {
		view.tags = tag_key(board);
		if (board == config.STAFF_BOARD)
			view.immortal = 1;
	}
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
		note_hash(m, msg.image.hash, msg.num);
	}
	m.hmset(key, view);
	m.set(key + ':body', body);

	if (msg.links) {
		m.hmset(key + ':links', msg.links);
		this.addBacklinks(m, num, op, msg.links);
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
	if (msg.links)
		view.links = msg.links;
	extract(view);
	delete view.ip;

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
					const n = post_volume(view, body);
					if (n > 0)
						update_throughput(m, ip, view.time, n);
					etc.augments.auth = {ip: ip};
				}
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
};

Y.addBacklinks = function(m, num, op, links) {
	for (let targetNum in links) {
		// Check if post exists through cache
		if (!(targetNum in OPs))
			continue;
		const key = (targetNum in TAGS ? 'thread' : 'post')
			+ `:${targetNum}:backlinks`;
		m.hset(key, num, op);
		this._log(m, links[targetNum], common.BACKLINK, [targetNum, num, op]);
	}
};

Y.remove_post = function (from_thread, num, callback) {
	num = parseInt(num);
	var op = OPs[num];
	if (!op)
		return callback(Muggle('No such post.'));
	if (op == num) {
		if (!from_thread)
			return callback('Deletion loop?!');
		return this.remove_thread(num, callback);
	}

	var r = this.connect();
	var self = this;
	if (from_thread) {
		var key = 'thread:' + op;
		r.lrem(key + ':posts', -1, num, function (err, delCount) {
			if (err)
				return callback(err);
			/* did someone else already delete this? */
			if (delCount != 1)
				return callback(null, -num);
			/* record deletion */
			r.rpush(key + ':dels', num, function (err) {
				if (err)
					winston.warn(err);
				gone_from_thread();
			});
		});
	}
	else
		gone_from_thread();

	function gone_from_thread() {
		var key = 'post:' + num;
		r.hset(key, 'hide', '1', function (err) {
			if (err) {
				/* Difficult to recover. Whatever. */
				winston.warn("Couldn't hide: " + err);
			}
			/* TODO push cache update? */
			delete OPs[num];

			callback(null, [op, num]);

			/* In the background, try to finish the post */
			self.finish_quietly(key, warn);
			self.hide_image(key, warn);
		});
	}
};

Y.remove_posts = function (nums, callback) {
	var self = this;
	tail.map(nums, this.remove_post.bind(this, true), all_gone);

	function all_gone(err, dels) {
		if (err)
			return callback(err);
		var threads = {}, already_gone = [];
		for (let i = 0, l = dels.length; i < l; i++) {
			let del = dels[i];
			if (Array.isArray(del)) {
				let op = del[0];
				if (!(op in threads))
					threads[op] = [];
				threads[op].push(del[1]);
			}
			else if (del < 0)
				already_gone.push(-del);
			else if (del)
				winston.warn('Unknown del: ' + del);
		}
		if (already_gone.length)
			winston.warn("Tried to delete missing posts: " +
					already_gone);
		if (_.isEmpty(threads))
			return callback(null);
		var m = self.connect().multi();
		for (let op in threads) {
			let nums = threads[op];
			nums.sort();
			self._log(m, op, common.DELETE_POSTS, nums, {
				cacheUpdate: {nums: nums}
			});
		}
		m.exec(callback);
	}
};

Y.remove_thread = function (op, callback) {
	if (OPs[op] != op)
		return callback(Muggle('Thread does not exist.'));
	var r = this.connect();
	var key = 'thread:' + op, dead_key = 'dead:' + op;
	var graveyardKey = 'tag:' + tag_key('graveyard');
	var etc = {cacheUpdate: {}};
	var self = this;
	async.waterfall([
	function (next) {
		get_all_replies(r, op, next);
	},
	function (nums, next) {
		etc.cacheUpdate.nums = nums;
		if (!nums || !nums.length)
			return next(null, []);
		tail.map(nums, self.remove_post.bind(self, false), next);
	},
	function (dels, next) {
		var m = r.multi();
		m.incr(graveyardKey + ':bumpctr');
		m.hgetall(key);
		m.exec(next);
	},
	function (rs, next) {
		var deadCtr = rs[0], post = rs[1];
		var tags = parse_tags(post.tags);
		/* Rename thread keys, move to graveyard */
		var m = r.multi();
		tags.forEach(function (tag) {
			var tagKey = tag_key(tag);
			m.zrem('tag:' + tagKey + ':threads', op);
		});
		m.zadd(graveyardKey + ':threads', deadCtr, op);
		etc.tags = tags;
		self._log(m, op, common.DELETE_THREAD, [], etc);
		m.hset(key, 'hide', 1);
		/* Next two vals are checked */
		m.renamenx(key, dead_key);
		m.renamenx(key + ':history', dead_key + ':history');
		m.exec(next);
	},
	function (results, done) {
		var dels = results.slice(-2);
		if (dels.some(function (x) { return x === 0; }))
			return done("Already deleted?!");
		removeOPTag(op);

		/* Extra renames now that we have renamenx exclusivity */
		var m = r.multi();
		m.rename(key + ':posts', dead_key + ':posts');
		m.rename(key + ':links', dead_key + ':links');
		m.exec(function (err) {
			done(err, null); /* second arg is remove_posts dels */
		});
		/* Background, might not even be there */
		self.finish_quietly(dead_key, warn);
		self.hide_image(dead_key, warn);
	}], callback);
};

// Purges all the thread's keys from the database and delete's all
// images contained
Y.purge_thread = function(op, board, callback) {
	let r = this.connect();
	const key = 'thread:' + op;
	let keysToDel = [],
		filesToDel = [],
		nums = [];
	async.waterfall([
		// Confirm thread can be deleted
		function(next) {
			let m = r.multi();
			m.exists(key);
			m.hget(key, 'immortal');
			m.exec(next);
		},
		function(res, next) {
			// Likely to happen, if interrupted mid-purge
			if (!res[0]) {
				r.zrem(`tag:${tag_key(board)}:threads`, op);
				return next(key + ' does not exist.');
			}
			if (parseInt(res[1], 10))
				return next(key + ' is immortal.');
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
			const imageTypes = ['src', 'thumb', 'mid'];
			let path = imager.media_path;
			for (let i = 0, l = res.length; i < l; i += 5) {
				const hash = res[i],
					key = posts.shift();
				if (!hash)
					continue;
				// Add links
				keysToDel.push(key);
				// `key:links` exists
				if (res[i + 1])
					keysToDel.push(key + ':links');
				if (res[i + 2])
					keysToDel.push(key + ':backlinks');
				if (res[i + 3])
					keysToDel.push(key + ':body');
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
};

/* BOILERPLATE CITY */

Y.remove_images = function (nums, callback) {
	if (config.READ_ONLY)
		return callback(Muggle("Read-only right now."));
	var threads = {};
	var rem = this.remove_image.bind(this, threads);
	var self = this;
	tail.forEach(nums, rem, function (err) {
		if (err)
			return callback(err);
		var m = self.connect().multi();
		for (let op in threads) {
			self._log(m, op, common.DELETE_IMAGES, threads[op], {
				tags: tags_of(op)
			});
		}
		m.exec(callback);
	});
};

Y.remove_image = function (threads, num, callback) {
	if (config.READ_ONLY)
		return callback(Muggle("Read-only right now."));
	var r = this.connect();
	var op = OPs[num];
	if (!op)
		callback(null, false);
	var key = (op == num ? 'thread:' : 'post:') + num;
	var self = this;
	r.hexists(key, 'src', function (err, exists) {
		if (err)
			return callback(err);
		if (!exists)
			return callback(null);
		self.hide_image(key, function (err) {
			if (err)
				return callback(err);
			r.hset(key, 'hideimg', 1, function (err, affected) {
				if (err)
					return callback(err);
				if (!affected)
					return callback(null);

				if (threads[op])
					threads[op].push(num);
				else
					threads[op] = [num];
				r.hincrby('thread:' + op, 'imgctr', -1, callback);
			});
		});
	});
};

Y.hide_image = function (key, callback) {
	if (config.READ_ONLY)
		return callback(Muggle("Read-only right now."));
	var r = this.connect();
	var hash;
	var imgKeys = ['hideimg', 'hash', 'src', 'thumb', 'mid'];
	r.hmget(key, imgKeys, move_dead);

	function move_dead(err, rs) {
		if (err)
			return callback(err);
		if (!rs)
			return callback(null);
		var info = {};
		for (let i = 0; i < rs.length; i++)
			info[imgKeys[i]] = rs[i];
		if (info.hideimg) /* already gone */
			return callback(null);
		hooks.trigger("buryImage", info, callback);
	}
};

Y.force_image_spoilers = function (nums, callback) {
	if (config.READ_ONLY)
		return callback(Muggle("Read-only right now."));
	var threads = {};
	var rem = this.spoiler_image.bind(this, threads);
	var self = this;
	tail.forEach(nums, rem, function (err) {
		if (err)
			return callback(err);
		var m = self.connect().multi();
		for (let op in threads) {
			self._log(m, op, common.SPOILER_IMAGES, threads[op], {
				tags: tags_of(op)
			});
		}
		m.exec(callback);
	});
};

Y.spoiler_image = function (threads, num, callback) {
	if (config.READ_ONLY)
		return callback(Muggle("Read-only right now."));
	var r = this.connect();
	var op = OPs[num];
	if (!op)
		callback(null, false);
	var key = (op == num ? 'thread:' : 'post:') + num;
	var spoilerKeys = ['src', 'spoiler'];
	r.hmget(key, spoilerKeys, function (err, info) {
		if (err)
			return callback(err);
		/* no image or already spoilt */
		if (!info[0] || info[1] || info[2])
			return callback(null);
		var index = common.pick_spoiler(-1).index;
		r.hmset(key, 'spoiler', index, function (err) {
			if (err)
				return callback(err);

			if (threads[op])
				threads[op].push([num, index]);
			else
				threads[op] = [[num, index]];
			callback(null);
		});
	});
};

Y.toggle_thread_lock = function (op, callback) {
	if (config.READ_ONLY)
		return callback(Muggle("Read-only right now."));
	if (OPs[op] != op)
		return callback(Muggle('Thread does not exist.'));
	var r = this.connect();
	var key = 'thread:' + op;
	var self = this;
	r.hexists(key, 'locked', function (err, locked) {
		if (err)
			return callback(err);
		var m = r.multi();
		if (locked)
			m.hdel(key, 'locked');
		else
			m.hset(key, 'locked', '1');
		var act = locked ? common.UNLOCK_THREAD : common.LOCK_THREAD;
		self._log(m, op, act, []);
		m.exec(callback);
	});
};

/* END BOILERPLATE CITY */

function warn(err) {
	if (err)
		winston.warn('Warning: ' + err);
}

Y.check_thread_locked = function (op, callback) {
	this.connect().hexists('thread:' + op, 'locked', function (err, lock) {
		if (err)
			callback(err);
		else
			callback(lock ? Muggle('Thread is locked.') : null);
	});
};

Y.check_throttle = function (ip, callback) {
	// So we can spam new threads in debug mode
	if (config.DEBUG)
		return callback(null);
	var key = 'ip:' + ip + ':throttle:thread';
	this.connect().exists(key, function (err, exists) {
		if (err)
			callback(err);
		else
			callback(exists ? Muggle('Too soon.') : null);
	});
};

function note_hash(m, hash, num) {
	m.zadd('imageDups',
		Date.now() + (config.DEBUG ? 30000 : 3600000),
		num + ':' + hash
	);
}

Y.add_image = function (post, alloc, ip, callback) {
	var r = this.connect();
	var num = post.num, op = post.op;
	if (!op)
		return callback(Muggle("Can't add another image to an OP."));
	var image = alloc.image;
	if (!image.pinky)
		return callback(Muggle("Image is wrong size."));
	delete image.pinky;

	var key = 'post:' + num;
	r.exists(key, function (err, exists) {
		if (err)
			return callback(err);
		if (!exists)
			return callback(Muggle("Post does not exist."));

		imager.commit_image_alloc(alloc, function (err) {
			if (err)
				return callback(err);
			add_it();
		});
	});

	var self = this;
	function add_it() {
		var m = r.multi();
		note_hash(m, image.hash, post.num);
		m.hmset(key, image);
		m.hincrby('thread:' + op, 'imgctr', 1);

		delete image.hash;
		self._log(m, op, common.INSERT_IMAGE, [num, image]);

		var now = Date.now();
		update_throughput(m, ip, now, post_volume({image: true}));
		m.exec(callback);
	}
};

Y.append_post = function (post, tail, old_state, extra, cb) {
	var m = this.connect().multi();
	var key = (post.op ? 'post:' : 'thread:') + post.num;
	/* Don't need to check .exists() thanks to client state */
	m.append(key + ':body', tail);
	/* XXX: fragile */
	if (old_state[0] != post.state[0] || old_state[1] != post.state[1])
		m.hset(key, 'state', post.state.join());
	if (extra.ip) {
		var now = Date.now();
		update_throughput(m, extra.ip, now, post_volume(null, tail));
	}
	if (!_.isEmpty(extra.new_links))
		m.hmset(key + ':links', extra.new_links);

	// possibly attach data for dice rolls etc. to the update
	var attached = {post: post, extra: extra, writeKeys: {}, attach: {}};
	var self = this;
	hooks.trigger("attachToPost", attached, function (err, attached) {
		if (err)
			return cb(err);
		for (let h in attached.writeKeys)
			m.hset(key, h, attached.writeKeys[h]);
		const num = post.num,
			op = post.op || num;
		var msg = [num, tail];
		var links = extra.links || {};
		if (extra.links)
			self.addBacklinks(m, num, op, links);

		var a = old_state[0], b = old_state[1];
		// message tail is [... a, b, links, attachment]
		// default values [... 0, 0, {}, {}] don't need to be sent
		// to minimize log output
		if (!_.isEmpty(attached.attach))
			msg.push(a, b, links, attached.attach);
		else if (!_.isEmpty(links))
			msg.push(a, b, links);
		else if (b)
			msg.push(a, b);
		else if (a)
			msg.push(a);

		self._log(m, op, common.UPDATE_POST, msg);
		m.exec(cb);
	});
};

function finish_off(m, key, body) {
	m.hset(key, 'body', body);
	m.del(key.replace('dead', 'thread') + ':body');
	m.hdel(key, 'state');
	m.srem('liveposts', key);
}

Y.finish_post = function (post, callback) {
	var m = this.connect().multi();
	var key = (post.op ? 'post:' : 'thread:') + post.num;
	/* Don't need to check .exists() thanks to client state */
	finish_off(m, key, post.body);
	this._log(m, post.op || post.num, common.FINISH_POST, [post.num]);
	m.exec(callback);
};

Y.finish_quietly = function (key, callback) {
	var r = this.connect();
	r.hexists(key, 'body', function (err, exists) {
		if (err)
			return callback(err);
		if (exists)
			return callback(null);
		var body_key = key.replace('dead', 'thread') + ':body';
		r.get(body_key, function (err, body) {
			if (err)
				return callback(err);
			var m = r.multi();
			finish_off(m, key, body);
			m.exec(callback);
		});
	});
};

Y.finish_all = function (callback) {
	var r = this.connect();
	var self = this;
	r.smembers('liveposts', function (err, keys) {
		if (err)
			return callback(err);
		async.forEach(keys, function (key, cb) {
			var m = r.multi();
			m.get(key + ':body');
			var isPost = key.slice(0, 5) == 'post:';
			if (isPost)
				m.hget(key, 'op');
			m.exec(function (err, rs) {
				if (err)
					return cb(err);
				var m = r.multi();
				finish_off(m, key, rs[0]);
				var n = parseInt(key.match(/:(\d+)$/)[1]);
				var op = isPost ? parseInt(rs[1], 10) : n;
				self._log(m, op, common.FINISH_POST, [n]);
				m.srem('liveposts', key);
				m.exec(cb);
			});
		}, callback);
	});
};

Y._log = function (m, op, kind, msg, opts) {
	opts = opts || {};
	msg = JSON.stringify(msg).slice(1, -1);
	msg = msg.length ? (kind + ',' + msg) : ('' + kind);
	winston.info("Log: " + msg);
	if (!op)
		throw new Error('No OP.');
	var key = 'thread:' + op;

	if (common.is_pubsub(kind)) {
		m.rpush(key + ':history', msg);
		m.hincrby(key, 'hctr', 1);
	}

	var opBit = op + ',';
	var len = opBit.length + msg.length;
	msg = len + '|' + opBit + msg;

	if (!_.isEmpty(opts.augments))
		msg += JSON.stringify(opts.augments);
	m.publish(key, msg);
	var tags = opts.tags || (this.tag ? [this.tag] : []);
	for (let i = 0, l = tags.length; i < l; i++) {
		m.publish('tag:' + tags[i], msg);
	}
	if (opts.cacheUpdate) {
		var info = {kind: kind, tag: tags[0], op: op};
		_.extend(info, opts.cacheUpdate);
		m.publish('cache', JSON.stringify(info));
	}
};

Y.fetch_backlogs = function (watching, callback) {
	var r = this.connect();
	var combined = [];
	forEachInObject(watching, function (thread, cb) {
		if (thread == 'live')
			return cb(null);
		var key = 'thread:' + thread + ':history';
		var sync = watching[thread];
		r.lrange(key, sync, -1, function (err, log) {
			if (err)
				return cb(err);

			var prefix = thread + ',';
			for (let i = 0, l = log.length; i < l; i++) {
				combined.push(prefix + log[i]);
			}

			cb(null);
		});
	}, function (errs) {
		callback(errs, combined);
	});
};

Y.get_post_op = function (num, callback) {
	var r = this.connect();
	r.hget('post:' + num, 'op', function (err, op) {
		if (err)
			return callback(err);
		else if (op)
			return callback(null, num, op);
		r.exists('thread:' + num, function (err, exists) {
			if (err)
				callback(err);
			else if (!exists)
				callback(null, null, null);
			else
				callback(null, num, num);
		});
	});
};

Y.get_tag = function(page) {
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
};

Y._get_each_thread = function(reader, ix, nums, catalog) {
	if (!nums || ix >= nums.length) {
		this.emit('end');
		reader.removeAllListeners('endthread');
		reader.removeAllListeners('end');
		return;
	}

	var self = this;
	function next_please () {
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
};

/* LURKERS */

class Reader extends events.EventEmitter {
	constructor(ident) {
		// Call the EventEmitter's constructor
		super();
		if (caps.can_moderate(ident))
			this.showIPs = true;
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
						m.hgetall(key + ':links');
						m.hgetall(key + ':backlinks');
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
						const links = rs.shift();
						if (links)
							opPost.links = links;
						const backlinks = rs.shift();
						if (backlinks)
							opPost.backlinks = backlinks;
						if (abbrev)
							total += parseInt(rs.shift(), 10);
						if (opts.showDead) {
							deadNums = rs.shift();
							if (abbrev)
								total += parseInt(rs.shift(), 10);
						}

						extract(opPost);
						opPost.omit = Math.max(total - abbrev, 0);
						if (!self.showIPs)
							delete opPost.ip;
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
				m.hgetall(key + ':links');
				m.hgetall(key + ':backlinks');
				m.exec(next);
			},
			function (data, next) {
				let pre_post = data[0];
				const links = data[1],
					backlinks = data[2];
				if (links)
					pre_post.links = links;
				if (backlinks)
					pre_post.backlinks = backlinks;
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
				if (post)
					extract(post);
				if (!self.showIPs)
					delete post.ip;
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

/* AMUSEMENT */

Y.get_fun = function (op, callback) {
	if (cache.funThread && op == cache.funThread) {
		/* Don't cache, for extra fun */
		fs.readFile('client/fun.js', 'UTF-8', callback);
	}
	else
		callback(null);
};

Y.set_fun_thread = function (op, callback) {
	if (OPs[op] != op)
		return callback(Muggle("Thread not found."));
	var self = this;
	fs.readFile('client/fun.js', 'UTF-8', function (err, funJs) {
		if (err)
			return callback(err);
		cache.funThread = op;
		var m = self.connect().multi();
		self._log(m, op, common.EXECUTE_JS, [funJs]);
		m.exec(callback);
	});
};

Y.get_banner = function (cb) {
	this.connect().get('banner:info', cb);
};

Y.set_banner = function (message, cb) {
	var r = this.connect();

	var self = this;
	r.set('banner:info', message, function(err) {
		if (err)
			return cb(err);

		// Dispatch new banner
		var m = r.multi();
		self._log(m, 0, common.UPDATE_BANNER, [message]);
		m.exec(cb);
	});
};

Y.get_current_body = function (num, cb) {
	var key = (OPs[num] == num ? 'thread:' : 'post:') + num;
	var m = this.connect().multi();
	m.hmget(key, 'hide', 'body');
	m.get(key + ':body');
	m.exec(function (err, rs) {
		if (err)
			return cb(err);
		var hide = rs[0][0], finalBody = rs[0][1];
		var liveBody = rs[1];
		if (hide)
			return cb(null);
		if (finalBody)
			return cb(null, finalBody, true);
		cb(null, liveBody || '', false);
	});
};

/* HELPERS */

function get_all_replies(r, op, cb) {
	var key = 'thread:' + op;
	r.lrange(key + ':posts', 0, -1, function(err, nums) {
		if (err)
			return cb(err);
		return cb(null, nums);
	});
}

function extract(post) {
	post.num = parseInt(post.num, 10);
	hooks.trigger_sync('extractPost', post);
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

function hmget_obj(r, key, keys, cb) {
	r.hmget(key, keys, function (err, rs) {
		if (err)
			return cb(err);
		var result = {};
		for (let i = 0; i < keys.length; i++)
			result[keys[i]] = rs[i];
		cb(null, result);
	});
}
