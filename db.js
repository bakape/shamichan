var _ = require('underscore'),
    async = require('async'),
    cache = require('./server/state').dbCache,
    caps = require('./server/caps'),
    common = require('./common'),
    config = require('./config'),
    events = require('events'),
    fs = require('fs'),
    hooks = require('./hooks'),
    hot = require('./server/state').hot,
    imager = require('./imager'),
    Muggle = require('./etc').Muggle,
    tail = require('./tail'),
    util = require('util'),
    winston = require('winston');

var imager = require('./imager'); /* set up hooks */

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
};

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
	var k = this.k;
	k.removeAllListeners('subscribe');
	k.on('message', this.on_message.bind(this));
	k.removeAllListeners('error');
	k.on('error', this.sink_sub.bind(this));
	this.subscription_callbacks.forEach(function (cb) {
		cb(null);
	});
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
		var modified = inject_extra(op, kind, msg, extra);
		// currently this won't modify op or kind,
		// but will have to watch out for that if that changes
		if (modified)
			msg = modified;
	}
	this.emit('update', op, kind, '[[' + msg + ']]');
};

S.on_sub_error = function (err) {
	winston.error("Subscription error:", (err.stack || err));
	this.commit_sudoku();
	this.subscription_callbacks.forEach(function (cb) {
		cb(err);
	});
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

function inject_extra(op, kind, msg, extra) {
	// Just one kind of insertion right now
	if (kind == common.INSERT_POST && extra.ip) {
		var m = msg.match(/^(\d+,\d+,\d+,)(.+)$/);
		var post = JSON.parse(m[2]);
		post.ip = extra.ip;
		return m[1] + JSON.stringify(post);
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
};
exports.OP_has_tag = OP_has_tag;

exports.first_tag_of = function (op) {
	var tags = TAGS[op];
	if (tags === undefined)
		return false;
	else if (typeof tags == 'number')
		return config.BOARDS[tags];
	else
		return config.BOARDS[tags[0]];
};

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
	else if (kind == common.MOVE_THREAD) {
		set_OP_tag(tag, op);
	}
	else if (kind == common.DELETE_POSTS) {
		msg.nums.forEach(function (num) {
			delete OPs[num];
		});
	}
	else if (kind == common.DELETE_THREAD) {
		msg.nums.forEach(function (num) {
			delete OPs[num];
		});
		delete TAGS[op];
	}
}

exports.track_OPs = function (callback) {
	var k = redis_client();
	k.subscribe('cache');
	k.once('subscribe', function () {
		load_OPs(callback);
	});
	k.on('message', update_cache);
	/* k persists for the purpose of cache updates */
};

exports.on_pub = function (name, handler) {
	// TODO: share redis connection
	var k = redis_client();
	k.subscribe(name);
	k.on('message', handler);
	/* k persists */
};

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
				if (!config.READ_ONLY && config.THREAD_EXPIRY
							&& tag != 'archive') {
					ps.push(refresh_expiry.bind(null,
							tag, op));
				}
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
			posts.forEach(function (num) {
				OPs[parseInt(num, 10)] = op;
			});
			cb(null);
		});
	}

	var expiryKey = expiry_queue_key();
	function refresh_expiry(tag, op, cb) {
		if (tag == config.STAFF_BOARD)
			return cb(null);
		var entry = op + ':' + tag_key(tag);
		var queries = ['time', 'immortal'];
		hmget_obj(r, 'thread:'+op, queries, function (err, thread) {
			if (err)
				return cb(err);
			if (!thread.time) {
				winston.warn('Thread '+op+" doesn't exist.");
				var m = r.multi();
				m.zrem(threadsKey, op);
				m.zrem(expiryKey, entry);
				m.exec(cb);
				return;
			}
			if (thread.immortal)
				return r.zrem(expiryKey, entry, cb);
			var score = expiry_queue_score(thread.time, tag);
			r.zadd(expiryKey, score, entry, cb);
		});
	}
}

function expiry_queue_score(time, board) {
	// Use default of 7 days, if not configured
	var expiry = config.THREAD_EXPIRY[board] || 3600 * 24 * 7;
	return Math.floor(parseInt(time, 10)/1000 + expiry);
}

function expiry_queue_key() {
	return 'expiry:all';
}
exports.expiry_queue_key = expiry_queue_key;

/* SOCIETY */

exports.is_board = function (board) {
	return config.BOARDS.indexOf(board) >= 0;
};

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
	this.removeAllListeners('end');
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
	for (var k in obj) {
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
	var self = this;
	this.subs.forEach(function (key) {
		var sub = SUBS[key];
		if (sub) {
			sub.removeListener('update', self.on_update);
			sub.removeListener('error', self.on_sink);
			if (sub.listeners('update').length == 0)
				sub.has_no_listeners();
		}
	});
	this.subs = [];
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
	var r = this.connect();
	if (!this.tag)
		return callback(Muggle("Can't retrieve board for posting."));
	var ip = extra.ip, board = extra.board, num = msg.num, op = msg.op;
	if (!num)
		return callback(Muggle("No post num."));
	else if (!ip)
		return callback(Muggle("No IP."));
	else if (op) {
		if (OPs[op] != op || !OP_has_tag(board, op)) {
			delete OPs[num];
			return callback(Muggle('Thread does not exist.'));
		}
	}

	var view = {time: msg.time, num: num, board: board, ip: ip, state: msg.state.join()};
	optPostFields.forEach(function (field) {
		if (msg[field])
			view[field] = msg[field];
	});
	var tagKey = 'tag:' + tag_key(this.tag);
	if (op)
		view.op = op;
	else {
		view.tags = tag_key(board);
		if (board == config.STAFF_BOARD)
			view.immortal = 1;
	}

	if (extra.image_alloc) {
		msg.image = extra.image_alloc.image;
		if (!op == msg.image.pinky)
			return callback(Muggle("Image is the wrong size."));
		delete msg.image.pinky;
	}

	var key = (op ? 'post:' : 'thread:') + num;
	var bump = !op || !common.is_sage(view.email);
	var m = r.multi();
	m.incr(tagKey + ':postctr'); // must be first
	if (op)
		m.hget('thread:' + op, 'subject'); // must be second
	if (bump)
		m.incr(tagKey + ':bumpctr');
	m.sadd('liveposts', key);

	hooks.trigger_sync('inlinePost', {src: msg, dest: view});
	if (msg.image) {
		if (op)
			m.hincrby('thread:' + op, 'imgctr', 1);
		else
			view.imgctr = 1;
		note_hash(m, msg.image.hash, msg.num);
	}
	m.hmset(key, view);
	m.set(key + ':body', body);
	if (msg.links)
		m.hmset(key + ':links', msg.links);

	var etc = {augments: {}, cacheUpdate: {}};
	if (op) {
		etc.cacheUpdate.num = num;
		var pre = 'thread:' + op;
		m.rpush(pre + ':posts', num);
	}
	else {
		// TODO: Add to alternate thread list?
		// set conditional hide?
		op = num;
		if (!view.immortal) {
			var score = expiry_queue_score(msg.time, board);
			var entry = num + ':' + tag_key(this.tag);
			m.zadd(expiry_queue_key(), score, entry);
		}
		/* Rate-limit new threads */
		if (ip != '127.0.0.1')
			m.setex('ip:'+ip+':throttle:thread',
					config.THREAD_THROTTLE, op);
	}

	/* Denormalize for backlog */
	view.nonce = msg.nonce;
	view.body = body;
	if (msg.links)
		view.links = msg.links;
	extract(view);
	delete view.ip;

	var self = this;
	async.waterfall([
	function (next) {
		if (!msg.image)
			return next(null);

		imager.commit_image_alloc(extra.image_alloc, next);
	},
	function (next) {
		if (ip) {
			var n = post_volume(view, body);
			if (n > 0)
				update_throughput(m, ip, view.time, n);
			etc.augments.auth = {ip: ip};
		}

		self._log(m, op, common.INSERT_POST, [num, view], etc);

		m.exec(next);
	},
	function (results, next) {
		if (!bump)
			return next();
		var postctr = results[0];
		var subject = subject_val(op,
				op==num ? view.subject : results[1]);
		var m = r.multi();
		m.zadd(tagKey + ':threads', postctr, op);
		if (subject)
			m.zadd(tagKey + ':subjects', postctr, subject);
		m.exec(next);
	}],
	function (err) {
		if (err) {
			delete OPs[num];
			return callback(err);
		}
		callback(null);
	});
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
		dels.forEach(function (del) {
			if (Array.isArray(del)) {
				var op = del[0];
				if (!(op in threads))
					threads[op] = [];
				threads[op].push(del[1]);
			}
			else if (del < 0)
				already_gone.push(-del);
			else if (del)
				winston.warn('Unknown del: ' + del);
		});
		if (already_gone.length)
			winston.warn("Tried to delete missing posts: " +
					already_gone);
		if (_.isEmpty(threads))
			return callback(null);
		var m = self.connect().multi();
		for (var op in threads) {
			var nums = threads[op];
			nums.sort();
			var opts = {cacheUpdate: {nums: nums}};
			self._log(m, op, common.DELETE_POSTS, nums, opts);
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
		var subject = subject_val(op, post.subject);
		/* Rename thread keys, move to graveyard */
		var m = r.multi();
		var expiryKey = expiry_queue_key();
		tags.forEach(function (tag) {
			var tagKey = tag_key(tag);
			m.zrem(expiryKey, op + ':' + tagKey);
			m.zrem('tag:' + tagKey + ':threads', op);
			if (subject)
				m.zrem('tag:' + tagKey + ':subjects', subject);
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

// Purges all the thread's keys from the database and delete's all images contained
Y.purge_thread = function(op, callback){
	var r = this.connect();
	var key = 'thread:' + op;
	async.waterfall([
		// Confirm thread can be deleted
		function(next){
			var m = r.multi();
			m.exists(key);
			m.hget(key, 'immortal');
			m.exec(next);
		},
		function(res, next){
			if (!res[0])
				return callback(Muggle(key + ' does not exist.'));
			if (parseInt(res[1], 10))
				return callback(Muggle(key + ' is immortal.'));
			// Get post list
			r.lrange(key + ':posts', 0, -1, next);
		},
		// Read all thread's hashes
		function(res, next){
			var m = r.multi();
			m.hgetall(key);
			if (res){
				res.forEach(function(el){
					m.hgetall('post:' + el);
				});
			}
			m.exec(next);
		},
		function(res, next){
			// Delete images
			var to_delete = [];
			var imp = imager.media_path;
			var m = r.multi();
			for (i = 0; i < res.length; i++){
				if (res[i].src)
					to_delete.push(imp('src', res[i].src));
				if (res[i].thumb)
					to_delete.push(imp('thumb', res[i].thumb));
				if (res[i].mid)
					to_delete.push(imp('mid', res[i].mid));
			}
			to_delete.forEach(function(el){
				fs.unlink(el, function(err){
					if (err)
						winston.error(err);
				});
			});
			m.lrange(key + ':posts', 0, -1);
			m.zrem('tag:' + res[0].tags + ':threads', op);
			m.exec(next);
		},
		function(res, done){
			// Delete post keys
			var m = r.multi();
			if (res[0]){
				res[0].forEach(function(entry){
					m.del('post:' + entry);
					m.del('post:' + entry + ':links');
				});
			}
			// Delete thread keys
			m.del(key);
			m.del(key + ':links');
			m.del(key + ':dels');
			m.del(key + ':history');
			m.del(key + ':posts');
			m.del(key + ':body');
			m.exec(done);
			removeOPTag(op);
		},
		callback
	]);
};

Y.archive_thread = function (op, callback) {
	var r = this.connect();
	var key = 'thread:' + op, archiveKey = 'tag:' + tag_key('archive');
	var self = this;
	async.waterfall([
	function (next) {
		var m = r.multi();
		m.exists(key);
		m.hget(key, 'immortal');
		m.zscore('tag:' + tag_key('graveyard') + ':threads', op);
		m.exec(next);
	},
	function (rs, next) {
		if (!rs[0])
			return callback(Muggle(key + ' does not exist.'));
		if (parseInt(rs[1], 10))
			return callback(Muggle(key + ' is immortal.'));
		if (rs[2])
			return callback(Muggle(key + ' is already deleted.'));
		var m = r.multi();
		// order counts
		m.hgetall(key);
		m.hgetall(key + ':links');
		m.llen(key + ':posts');
		m.lrange(key + ':dels', 0, -1);
		m.exec(next);
	},
	function (rs, next) {
		var view = rs[0], links = rs[1], replyCount = rs[2], dels = rs[3];
		var subject = subject_val(op, view.subject);
		var tags = view.tags;
		var m = r.multi();
		// move to archive tag only
		m.hset(key, 'origTags', tags);
		m.hset(key, 'tags', tag_key('archive'));
		tags = parse_tags(tags);
		tags.forEach(function (tag) {
			var tagKey = 'tag:' + tag_key(tag);
			m.zrem(tagKey + ':threads', op);
			if (subject)
				m.zrem(tagKey + ':subjects', subject);
		});
		m.zadd(archiveKey + ':threads', op, op);
		self._log(m, op, common.DELETE_THREAD, [], {tags: tags});

		// shallow thread insertion message in archive
		if (!_.isEmpty(links))
			view.links = links;
		extract(view);
		delete view.ip;
		view.replyctr = replyCount;
		view.hctr = 0;
		var etc = {tags: ['archive'], cacheUpdate: {}};
		self._log(m, op, common.MOVE_THREAD, [view], etc);

		// clear history; note new history could be added
		// for deletion in the archive
		// (a bit silly right after adding a new entry)
		m.hdel(key, 'hctr');
		m.del(key + ':history');

		// delete hidden posts
		dels.forEach(function (num) {
			m.del('post:' + num);
			m.del('post:' + num + ':links');
		});
		m.del(key + ':dels');

		m.exec(next);
	},
	function (results, done) {
		set_OP_tag(config.BOARDS.indexOf('archive'), op);
		done();
	}], callback);
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
		for (var op in threads)
			self._log(m, op, common.DELETE_IMAGES, threads[op],
					{tags: tags_of(op)});
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
				r.hincrby('thread:' + op, 'imgctr', -1,
						callback);
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
		for (var i = 0; i < rs.length; i++)
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
		for (var op in threads)
			self._log(m, op, common.SPOILER_IMAGES, threads[op],
					{tags: tags_of(op)});
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
	var self = this;
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
	var key = 'ip:' + ip + ':throttle:thread';
	this.connect().exists(key, function (err, exists) {
		if (err)
			callback(err);
		else
			callback(exists ? Muggle('Too soon.') : null);
	});
};

function note_hash(m, hash, num) {
	m.zadd('imageDups', Date.now() + (config.DEBUG ? 30000 : 3600000), num + ':' + hash);
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
		var n = post_volume({image: true});
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
		for (var h in attached.writeKeys)
			m.hset(key, h, attached.writeKeys[h]);
		var msg = [post.num, tail];
		var links = extra.links || {};

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

		self._log(m, post.op || post.num, common.UPDATE_POST, msg);
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
	tags.forEach(function (tag) {
		m.publish('tag:' + tag, msg);
	});

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
			log.forEach(function (entry) {
				combined.push(prefix + entry);
			});

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

Y.get_tag = function (page) {
	var r = this.connect();
	var self = this;
	var key = 'tag:' + tag_key(this.tag) + ':threads';
	var reverseOrder = this.tag == 'archive';
	if (page < 0 && !reverseOrder)
		page = 0;
	var start = page * hot.THREADS_PER_PAGE;
	var end = start + hot.THREADS_PER_PAGE - 1;
	var m = r.multi();
	if (reverseOrder)
		m.zrange(key, start, end);
	else
		m.zrevrange(key, start, end);
	m.zcard(key);
	m.exec(function (err, res) {
		if (err)
			return self.emit('error', err);
		var nums = res[0];
		if (page > 0 && !nums.length)
			return self.emit('nomatch');
		if (reverseOrder)
			nums.reverse();
		self.emit('begin', res[1]);
		var reader = new Reader(self);
		reader.on('error', self.emit.bind(self, 'error'));
		reader.on('thread', self.emit.bind(self, 'thread'));
		reader.on('post', self.emit.bind(self, 'post'));
		reader.on('endthread', self.emit.bind(self, 'endthread'));
		self._get_each_thread(reader, 0, nums);
	});
};

Y._get_each_thread = function (reader, ix, nums) {
	if (!nums || ix >= nums.length) {
		this.emit('end');
		reader.removeAllListeners('endthread');
		reader.removeAllListeners('end');
		return;
	}
	var self = this;
	var next_please = function () {
		reader.removeListener('end', next_please);
		reader.removeListener('nomatch', next_please);
		self._get_each_thread(reader, ix+1, nums);
	};
	reader.on('end', next_please);
	reader.on('nomatch', next_please);
	reader.get_thread(this.tag, nums[ix], {
			abbrev: hot.ABBREVIATED_REPLIES || 5
	});
};

/* LURKERS */

function Reader(yakusoku) {
	events.EventEmitter.call(this);
	this.y = yakusoku;
}

util.inherits(Reader, events.EventEmitter);
exports.Reader = Reader;

Reader.prototype.get_thread = function (tag, num, opts) {
	var r = this.y.connect();
	var graveyard = (tag == 'graveyard');
	if (graveyard)
		opts.showDead = true;
	var key = (graveyard ? 'dead:' : 'thread:') + num;
	var self = this;
	r.hgetall(key, function (err, pre_post) {
		if (err)
			return self.emit('error', err);
		if (_.isEmpty(pre_post)) {
			if (!opts.redirect)
				return self.emit('nomatch');
			r.hget('post:' + num, 'op',
						function (err, op) {
				if (err)
					self.emit('error', err);
				else if (!op)
					self.emit('nomatch');
				else
					self.emit('redirect', op);
			});
			return;
		}
		var exists = true;
		if (pre_post.hide && !opts.showDead)
			exists = false;
		var tags = parse_tags(pre_post.tags);
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
		pre_post.num = num;
		pre_post.time = parseInt(pre_post.time, 10);

		var nums, deadNums, opPost;
		var abbrev = opts.abbrev || 0, total = 0;
		async.waterfall([
		function (next) {
			with_body(r, key, pre_post, next);
		},
		function (fullPost, next) {
			opPost = fullPost;
			var m = r.multi();
			var postsKey = key + ':posts';

			// order is important!
			m.lrange(postsKey, -abbrev, -1);
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
			if (abbrev)
				total += parseInt(rs.shift(), 10);
			if (opts.showDead) {
				deadNums = rs.shift();
				if (abbrev)
					total += parseInt(rs.shift(), 10);
			}

			extract(opPost);
			next(null);
		}],
		function (err) {
			if (err)
				return self.emit('error', err);
			if (deadNums)
				nums = merge_posts(nums, deadNums, abbrev);
			var omit = Math.max(total - abbrev, 0);
			self.emit('thread', opPost, omit);
			self._get_each_reply(tag, 0, nums, opts);
		});
	});
};

function merge_posts(nums, deadNums, abbrev) {
	var i = nums.length - 1, pi = deadNums.length - 1;
	if (pi < 0)
		return nums;
	var merged = [];
	while (!abbrev || merged.length < abbrev) {
		if (i >= 0 && pi >= 0) {
			var num = nums[i], pNum = deadNums[pi];
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

Reader.prototype._get_each_reply = function (tag, ix, nums, opts) {
	if (!nums || ix >= nums.length) {
		this.emit('endthread');
		this.emit('end');
		return;
	}
	var num = parseInt(nums[ix], 10);
	var self = this;
	this.get_post('post', num, opts, function (err, post) {
		if (err)
			return self.emit('error', err);
		if (post)
			self.emit('post', post);
		self._get_each_reply(tag, ix + 1, nums, opts);
	});
};

Reader.prototype.get_post = function (kind, num, opts, cb) {
	var r = this.y.connect();
	var key = kind + ':' + num;
	var self = this;
	async.waterfall([
	function (next) {
		r.hgetall(key, next);
	},
	function (pre_post, next) {
		var exists = !(_.isEmpty(pre_post));
		if (exists && pre_post.hide && !opts.showDead)
			exists = false;
		if (!exists)
			return next(null, null);

		pre_post.num = num;
		pre_post.time = parseInt(pre_post.time, 10);
		if (kind == 'post') {
			pre_post.op = parseInt(pre_post.op, 10);
		}
		else {
			var tags = parse_tags(pre_post.tags);
			// TODO: filter by ident eligibility and attach
		}
		with_body(r, key, pre_post, next);
	},
	function (post, next) {
		if (post)
			extract(post);
		next(null, post);
	}], cb);
};

function get_all_replies(r, op, cb) {
	var key = 'thread:' + op;
	r.lrange(key + ':posts', 0, -1, function(err, nums) {
		if (err)
			return cb(err);
		return cb(null, nums);
	});
};

/* AUTHORITY */

function Filter(tag) {
	events.EventEmitter.call(this);
	this.tag = tag;
};

util.inherits(Filter, events.EventEmitter);
exports.Filter = Filter;
var F = Filter.prototype;

F.connect = function () {
	if (!this.r) {
		this.r = global.redis;
	}
	return this.r;
};

F.get_all = function (limit) {
	var self = this;
	var r = this.connect();
	r.zrange('tag:' + tag_key(this.tag) + ':threads', 0, -1, go);
	function go(err, threads) {
		if (err)
			return self.failure(err);
		async.forEach(threads, do_thread, self.check_done.bind(self));
	}
	function do_thread(op, cb) {
		var key = 'thread:' + op;
		r.llen(key + ':posts', function (err, len) {
			if (err)
				cb(err);
			len = parseInt(len);
			if (len > limit)
				return cb(null);
			var thumbKeys = ['thumb', 'src'];
			r.hmget(key, thumbKeys, function (err, rs) {
				if (err)
					cb(err);
				var thumb = rs[0] || rs[1] || rs[2];
				self.emit('thread', {num: op, thumb: thumb});
				cb(null);
			});
		});
	}
};

F.check_done = function (err) {
	if (err)
		this.failure(err);
	else
		this.success();
};

F.success = function () {
	this.emit('end');
	this.cleanup();
};

F.failure = function (err) {
	this.emit('error', err);
	this.cleanup();
};

F.cleanup = function () {
	this.removeAllListeners('error');
	this.removeAllListeners('thread');
	this.removeAllListeners('end');
};

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

Y.teardown = function (board, cb) {
	var m = this.connect().multi();
	var filter = new Filter(board);
	var self = this;
	filter.get_all(NaN); // no length limit
	filter.on('thread', function (thread) {
		self._log(m, thread.num, common.TEARDOWN, []);
	});
	filter.on('error', cb);
	filter.on('end', function () {
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

function extract(post) {
	hooks.trigger_sync('extractPost', post);
}

function with_body(r, key, post, callback) {
	if (post.body !== undefined)
		callback(null, post);
	else
		r.get(key + ':body', function (err, body) {
			if (err)
				return callback(err);
			if (body !== null) {
				post.body = body;
				post.editing = true;
				return callback(null, post);
			}
			// Race condition between finishing posts
			r.hget(key, 'body', function (err, body) {
				if (err)
					return callback(err);
				post.body = body;
				callback(null, post);
			});
		});
};

function subject_val(op, subject) {
	return subject && (op + ':' + subject);
}

var tag_key = exports.tag_key =  function(tag) {
	return tag.length + ':' + tag;
};

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
		for (var i = 0; i < keys.length; i++)
			result[keys[i]] = rs[i];
		cb(null, result);
	});
}

// Remove expired duplicate image hashes
function cleanUpDups(){
	global.redis.zremrangebyscore('imageDups', 0, Date.now());
}

setInterval(cleanUpDups, 60000);
