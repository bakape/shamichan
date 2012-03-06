var _ = require('./lib/underscore'),
    async = require('async'),
    cache = require('./server/state').dbCache,
    common = require('./common'),
    config = require('./config'),
    events = require('events'),
    fs = require('fs'),
    games = require('./server/games'),
    hooks = require('./hooks'),
    redis = require('redis'),
    util = require('util');

var OPs = exports.OPs = cache.OPs;
var TAGS = exports.TAGS = cache.opTags;
var SUBS = exports.SUBS = cache.threadSubs;

function redis_client() {
	return redis.createClient(config.REDIS_PORT || undefined);
}
exports.redis_client = redis_client;

/* REAL-TIME UPDATES */

function Subscription(target) {
	events.EventEmitter.call(this);
	this.setMaxListeners(0);
	this.target = target;
	this.subscription_callbacks = [];

	this.k = redis_client();
	this.k.on('error', this.on_sub_error.bind(this));
	this.k.on('subscribe', this.on_sub.bind(this));
	this.k.subscribe(target);
};

util.inherits(Subscription, events.EventEmitter);
var S = Subscription.prototype;

S.when_ready = function (cb) {
	if (this.subscription_callbacks)
		this.subscription_callbacks.push(cb);
	else
		cb(null);
};

S.on_sub = function () {
	var k = this.k;
	k.removeAllListeners('subscribe');
	k.on('message', this.on_message.bind(this));
	k.removeAllListeners('error');
	k.on('error', this.sink_sub.bind(this));
	this.subscription_callbacks.forEach(function (cb) {
		cb(null);
	});
	delete this.subscription_callbacks;
};

S.on_message = function (chan, msg) {
	var thread, info, kind, num;
	var m = chan.match(/^thread:(\d+)$/);
	if (m) {
		thread = parseInt(m[1], 10);
		info = msg.match(/^(\d+),(\d+)/);
		kind = parseInt(info[1], 10);
		num = parseInt(info[2], 10);
	}
	else {
		/* TEMP: Want to keep these uniform really... */
		info = msg.match(/^(\d+),(\d+),(\d+)/);
		thread = parseInt(info[1], 10);
		kind = parseInt(info[2], 10);
		num = parseInt(info[3], 10);
		/* BLEH */
		msg = msg.slice(info[1].length + 1);
	}
	this.emit('update', thread, num, kind, msg);
};

S.on_sub_error = function (err) {
	console.log("Subscription error:", err.stack || err); /* TEMP? */
	this.commit_sudoku();
	this.subscription_callbacks.forEach(function (cb) {
		cb(err);
	});
	delete this.subscription_callbacks;
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
	if (SUBS[this.target] === this)
		delete SUBS[this.target];
	this.removeAllListeners('update');
	this.removeAllListeners('error');
};

S.has_no_listeners = function () {
	/* Possibly idle out after a while */
	var self = this;
	setTimeout(function () {
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

exports.OP_has_tag = function (tag, op) {
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

exports.first_tag_of = function (op) {
	var tags = TAGS[op];
	if (tags === undefined)
		return false;
	else if (typeof tags == 'number')
		return config.BOARDS[tags];
	else
		return config.BOARDS[tags[0]];
};

function on_OP_message(pat, chan, msg) {
	// This sucks. Need to unify this damn format.
	var op, num, kind, tag;
	if (pat == 'tag:*') {
		var m = msg.match(/^(\d+),(\d+)/);
		op = num = parseInt(m[1], 10);
		kind = parseInt(m[2], 10);
		tag = chan.slice(4);
	}
	else {
		op = parseInt(chan.match(/^thread:(\d+)/)[1], 10);
		var m = msg.match(/^(\d+),(\d+)/);
		kind = parseInt(m[1], 10);
		num = parseInt(m[2], 10);
		tag = false;
	}

	if (kind == common.INSERT_POST) {
		if (tag)
			add_OP_tag(config.BOARDS.indexOf(tag), op);
		else
			OPs[num] = op;
	}
	else if (tag && kind == common.MOVE_THREAD) {
		set_OP_tag(config.BOARDS.indexOf(tag), op);
	}
}

exports.track_OPs = function (callback) {
	var k = redis_client();
	k.psubscribe('thread:*');
	k.once('psubscribe', function () {
		var r = redis_client();
		load_OPs(r, function (err) {
			r.quit();
			callback(err);
		});
		k.psubscribe('tag:*');
	});
	k.on('pmessage', on_OP_message);
};

function load_OPs(r, callback) {
	var boards = config.BOARDS;
	// Want consistent ordering in the TAGS entries for multi-tag threads
	// (so do them in series)
	async.forEachSeries(boards, scan_board, callback);

	function scan_board(tag, cb) {
		var tagIndex = boards.indexOf(tag);
		var key = 'tag:' + tag_key(tag);
		r.zrange(key + ':threads', 0, -1, function (err, threads) {
			if (err)
				return cb(err);
			async.forEach(threads, function (op, cb) {
				op = parseInt(op, 10);
				var ps = [scan_thread.bind(null,tagIndex,op)];
				if (config.THREAD_EXPIRY && tag != 'archive')
					ps.push(refresh_expiry.bind(null,
							tag, op));
				async.parallel(ps, cb);
			}, cb);
		});
	}

	function scan_thread(tagIndex, op, cb) {
		op = parseInt(op, 10);
		add_OP_tag(tagIndex, op);
		OPs[op] = op;
		var key = 'thread:' + op;
		r.lrange(key + ':posts', 0, -1, function (err, posts) {
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
		var entry = op + ':' + tag_key(tag);
		var queries = ['time', 'immortal'];
		hmget_obj(r, 'thread:'+op, queries, function (err, thread) {
			if (err)
				return cb(err);
			if (thread.immortal)
				return r.zrem(expiryKey, entry, cb);
			var score = expiry_queue_score(thread.time);
			r.zadd(expiryKey, score, entry, cb);
		});
	}
}

function expiry_queue_score(time) {
	return Math.floor(parseInt(time, 10)/1000 + config.THREAD_EXPIRY);
}

function expiry_queue_key() {
	return 'expiry:' + config.THREAD_EXPIRY;
}
exports.expiry_queue_key = expiry_queue_key;

/* SOCIETY */

exports.is_board = function (board) {
	return config.BOARDS.indexOf(board) >= 0;
};

function Yakusoku(board) {
	events.EventEmitter.call(this);
	this.id = ++(cache.YAKUMAN);
	this.tag = board;
}

util.inherits(Yakusoku, events.EventEmitter);
exports.Yakusoku = Yakusoku;
var Y = Yakusoku.prototype;

Y.connect = function () {
	if (!this.r) {
		this.r = redis_client();
		this.r.on('error', console.error.bind(console));
	}
	return this.r;
};

Y.disconnect = function () {
	if (this.r) {
		this.r.quit();
		this.r.removeAllListeners('error');
	}
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

Y.target_key = function (key) {
	return (key == 'live') ? 'tag:' + this.tag : 'thread:' + key;
};

Y.kiku = function (targets, on_update, on_sink, callback) {
	var self = this;
	this.on_update = on_update;
	this.on_sink = on_sink;
	forEachInObject(targets, function (target, cb) {
		var key = self.target_key(target);
		var sub = SUBS[key];
		if (!sub) {
			sub = new Subscription(key);
			SUBS[key] = sub;
		}
		sub.on('update', on_update);
		sub.on('error', on_sink);
		sub.when_ready(cb);
	}, callback);
};

Y.kikanai = function (targets) {
	for (var target in targets) {
		var sub = SUBS[this.target_key(target)];
		if (sub) {
			sub.removeListener('update', this.on_update);
			sub.removeListener('error', this.on_sink);
			if (sub.listeners('update').length == 0)
				sub.has_no_listeners();
		}
	}
};

function update_throughput(m, ip, when, quant) {
	var key = 'ip:' + ip + ':';
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
	var r = this.connect();
	if (ip == '127.0.0.1')
		return reserve();

	var key = 'ip:' + ip + ':';
	var now = new Date().getTime();
	var shortTerm = key + short_term_timeslot(now);
	var longTerm = key + long_term_timeslot(now);
	r.mget([shortTerm, longTerm], function (err, quants) {
		if (err) {
			console.error(err);
			return callback("Limiter failure.");
		}
		if (quants[0] > config.SHORT_TERM_LIMIT ||
				quants[1] > config.LONG_TERM_LIMIT)
			return callback('Reduce your speed.');

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

Y.insert_post = function (msg, body, extra, callback) {
	var r = this.connect();
	if (!this.tag)
		return callback("Can't retrieve board for posting.");
	var self = this;
	var ip = extra.ip, board = extra.board, num = msg.num, op = msg.op;
	if (!num)
		return callback("No post num.");
	else if (!ip)
		return callback("No IP.");
	else if (op && OPs[op] != op) {
		delete OPs[num];
		return callback('Thread does not exist.');
	}

	var view = {time: msg.time, ip: ip, state: msg.state.join()};
	var tagKey = 'tag:' + tag_key(this.tag);
	if (msg.name)
		view.name = msg.name;
	if (msg.trip)
		view.trip = msg.trip;
	if (msg.email)
		view.email = msg.email;
	if (msg.auth)
		view.auth = msg.auth;
	if (op)
		view.op = op;
	else
		view.tags = tag_key(board);

	var key = (op ? 'post:' : 'thread:') + num;
	var bump = !op || !common.is_sage(view.email);
	var m = r.multi();
	m.incr(tagKey + ':postctr');
	if (bump)
		m.incr(tagKey + ':bumpctr');
	inline(view, msg);
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
	if (op) {
		m.rpush('thread:' + op + ':posts', num);
	}
	else {
		op = num;
		var score = expiry_queue_score(msg.time);
		var entry = num + ':' + tag_key(this.tag);
		m.zadd(expiry_queue_key(), score, entry);
		/* Rate-limit new threads */
		if (ip != '127.0.0.1')
			m.setex('ip:' + ip, config.THREAD_THROTTLE, op);
	}

	/* Denormalize for backlog */
	view.nonce = msg.nonce;
	view.body = body;
	if (msg.links)
		view.links = msg.links;
	extract(view);
	delete view.ip;
	self._log(m, op, common.INSERT_POST, [num, view]);

	if (ip) {
		var len = (body ? body.length : 0) + (view.image
				? config.IMAGE_CHARACTER_WORTH : 0);
		if (len > 0)
			update_throughput(m, ip, view.time, len);
	}

	m.exec(function (err, results) {
		if (err) {
			delete OPs[num];
			return callback(err);
		}
		if (!bump)
			return callback(null);
		r.zadd(tagKey + ':threads', results[0], op,
					function (err) {
			if (err)
				console.error("Bump error: " + err);
			callback(null);
		});
	});
};

Y.remove_post = function (from_thread, num, callback) {
	num = parseInt(num);
	var op = OPs[num];
	if (!op)
		return callback('No such post.');
	if (op == num) {
		if (!from_thread)
			return callback('Deletion loop?!');
		return this.remove_thread(num, callback);
	}

	var r = this.connect();
	var self = this;
	if (from_thread)
		r.lrem('thread:' + op + ':posts', -1, num, gone_from_thread);
	else
		gone_from_thread(null, 1);

	function gone_from_thread(err, deleted) {
		if (err)
			return callback(err);
		if (deleted != 1)
			return callback(null, -num); /* already gone */
		var key = 'post:' + num;
		r.hset(key, 'hide', '1', function (err) {
			if (err) {
				/* Difficult to recover. Whatever. */
				console.error(err);
			}
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
	async.map(nums, this.remove_post.bind(this, true), all_gone);

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
				console.warn('Unknown del:', del);
		});
		if (already_gone.length)
			console.warn("Tried to delete missing posts: ",
					already_gone);
		if (_.isEmpty(threads))
			return callback(null);
		var m = self.connect().multi();
		for (var op in threads) {
			var nums = threads[op];
			nums.sort();
			self._log(m, op, common.DELETE_POSTS, nums);
		}
		m.exec(callback);
	}
};

Y.remove_thread = function (op, callback) {
	if (OPs[op] != op)
		return callback('Thread does not exist.');
	var r = this.connect();
	var key = 'thread:' + op, dead_key = 'dead:' + op;
	var graveyardKey = 'tag:' + tag_key('graveyard');
	var self = this;
	async.waterfall([
	function (next) {
		r.lrange(key + ':posts', 0, -1, next);
	},
	function (nums, next) {
		if (!nums || !nums.length)
			return next(null, []);
		async.map(nums, self.remove_post.bind(self, false), next);
	},
	function (dels, next) {
		var m = r.multi();
		m.incr(graveyardKey + ':bumpctr');
		m.hget(key, 'tags');
		m.exec(next);
	},
	function (rs, next) {
		var deadCtr = rs[0], tags = parse_tags(rs[1]);
		/* Rename thread keys, move to graveyard */
		var m = r.multi();
		var expiryKey = expiry_queue_key();
		tags.forEach(function (tag) {
			var tagKey = tag_key(tag);
			m.zrem(expiryKey, op + ':' + tagKey);
			m.zrem('tag:' + tagKey + ':threads', op);
		});
		m.zadd(graveyardKey + ':threads', deadCtr, op);
		self._log(m, op, common.DELETE_THREAD, [op], tags);
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
		delete OPs[op];
		delete TAGS[op];
		done();
		/* Background, might not even be there */
		var nop = function (err) {};
		r.renamenx(key + ':posts', dead_key + ':posts', nop);
		r.renamenx(key + ':links', dead_key + ':links', nop);
		self.finish_quietly(dead_key, warn);
		self.hide_image(dead_key, warn);
	}], callback);
};

Y.archive_thread = function (op, callback) {
	var r = this.connect();
	var key = 'thread:' + op, archiveKey = 'tag:' + tag_key('archive');
	var self = this;
	async.waterfall([
	function (next) {
		var m = r.multi();
		m.exists(key);
		m.zscore('tag:' + tag_key('graveyard') + ':threads', op);
		m.exec(next);
	},
	function (rs, next) {
		if (!rs[0])
			return callback('Thread does not exist.');
		if (rs[1])
			return callback('Thread is already deleted.');
		var m = r.multi();
		m.incr(archiveKey + ':bumpctr');
		m.hgetall(key);
		m.hgetall(key + ':links');
		m.llen(key + ':posts');
		m.exec(next);
	},
	function (rs, next) {
		var bumpCtr = rs[0], view = rs[1], links = rs[2],
				replyCount = rs[3];
		var tags = view.tags;
		var m = r.multi();
		// move to archive tag only
		m.hset(key, 'origTags', tags);
		m.hset(key, 'tags', tag_key('archive'));
		tags = parse_tags(tags);
		tags.forEach(function (tag) {
			m.zrem('tag:' + tag_key(tag) + ':threads', op);
		});
		m.zadd(archiveKey + ':threads', bumpCtr, op);
		self._log(m, op, common.DELETE_THREAD, [op], tags);

		// shallow thread insertion message in archive
		if (!_.isEmpty(links))
			view.links = msg.links;
		extract(view);
		delete view.ip;
		view.replyctr = replyCount;
		self._log(m, op, common.MOVE_THREAD, [op, view], ['archive']);

		m.exec(next);
	},
	function (results, done) {
		TAGS[op] = config.BOARDS.indexOf('archive');
		done();
	}], callback);
};

Y.hide_image = function (key, callback) {
	var r = this.connect();
	var hash;
	r.hmget(key, ['hash', 'src', 'thumb', 'realthumb'], move_dead);

	function move_dead(err, pics) {
		if (err)
			return callback(err);
		if (!pics)
			return callback(null);
		hash = pics[0];
		if (pics[1])
			require('./server/pix').bury_image(pics[1], pics[2],
					pics[3], free_hash);
		else
			free_hash(null);
	}

	function free_hash(err) {
		if (err)
			return callback(err);
		if (hash)
			r.del('hash:' + hash, callback);
		else
			callback(null);
	}
};

function warn(err) {
	if (err)
		console.warn('Warning: ' + err);
}

Y.check_throttle = function (ip, callback) {
	this.connect().exists('ip:' + ip, function (err, exists) {
		if (err)
			callback(err);
		else
			callback(exists ? 'Too soon.' : null);
	});
};

function note_hash(m, hash, num) {
	var key = 'hash:' + hash;
	m.setex(key, config.DEBUG ? 30 : 3600, num);
}

Y.check_duplicate = function (hash, callback) {
	this.connect().get('hash:'+hash, function (err, num) {
		if (err)
			callback(err);
		else if (num)
			callback('Duplicate of >>' + num + '.');
		else
			callback(false);
	});
};

Y.add_image = function (post, image, ip, callback) {
	var r = this.connect();
	var num = post.num, op = post.op;
	if (!op)
		return callback("Can't add another image to an OP.");
	var key = 'post:' + num;
	var self = this;
	r.exists(key, function (err, exists) {
		if (err)
			return callback(err);
		if (!exists)
			return callback("Post does not exist.");
		var m = r.multi();
		note_hash(m, image.hash, post.num);
		m.hmset(key, image);
		m.hincrby('thread:' + op, 'imgctr', 1);

		delete image.hash;
		self._log(m, op, common.INSERT_IMAGE, [num, image]);

		var now = new Date().getTime();
		update_throughput(m, ip, now, config.IMAGE_CHARACTER_WORTH);
		m.exec(callback);
	});
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
		var now = new Date().getTime();
		update_throughput(m, extra.ip, now, tail.length);
	}
	if (!_.isEmpty(extra.new_links))
		m.hmset(key + ':links', extra.new_links);
	if (extra.new_dice) {
		// Only need to update when new dice are rolled
		var flat = {};
		// This is a little janky... possibly change inline API?
		games.inline_dice(flat, post.dice);
		m.hset(key, 'dice', flat.dice);
	}
	var msg = [post.num, tail];
	var links = extra.links || {};
	if (extra.new_dice)
		msg.push(old_state[0], old_state[1], links,
				{dice: extra.new_dice});
	else if (extra.links)
		msg.push(old_state[0], old_state[1], links);
	else if (old_state[1])
		msg.push(old_state[0], old_state[1]);
	else if (old_state[0])
		msg.push(old_state[0]);
	this._log(m, post.op || post.num, common.UPDATE_POST, msg);
	m.exec(cb);
};

function finish_off(m, key, body) {
	m.hset(key, 'body', body);
	m.del(key.replace('dead', 'thread') + ':body');
	m.hdel(key, 'state');
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
	r.keys('*:body', function (err, keys) {
		if (err)
			return callback(err);
		async.forEach(keys, function (body_key, cb) {
			var key = body_key.slice(0, -5);
			var m = r.multi();
			m.get(body_key);
			if (key.slice(0, 5) == 'post:')
				m.hget(key, 'op');
			m.exec(function (err, rs) {
				if (err)
					return cb(err);
				m = r.multi();
				finish_off(m, key, rs[0]);
				var n = parseInt(key.match(/:(\d+)$/)[1]);
				var op = parseInt(rs[1]) || n;
				self._log(m, op, common.FINISH_POST, [n]);
				m.exec(cb);
			});
		}, callback);
	});
};

Y._log = function (m, op, kind, msg, tags) {
	msg.unshift(kind);
	msg = JSON.stringify(msg).slice(1, -1);
	console.log("Log:", msg);
	if (!op)
		throw new Error('No OP.');
	var key = 'thread:' + op;
	if (common.is_pubsub(kind)) {
		m.rpush(key + ':history', msg);
		m.hincrby(key, 'hctr', 1);
	}
	m.publish(key, msg);
	msg = op + ',' + msg;
	if (!tags)
		tags = this.tag ? [this.tag] : [];
	tags.forEach(function (tag) {
		m.publish('tag:' + tag, msg);
	});
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

			/* TEMP */
			log.forEach(function (entry) {
				combined.push(entry + ',' + thread);
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
}

Y.get_tag = function (page) {
	var r = this.connect();
	var self = this;
	var key = 'tag:' + tag_key(this.tag) + ':threads';
	var start = page * config.THREADS_PER_PAGE;
	var end = start + config.THREADS_PER_PAGE - 1;
	var m = r.multi();
	m.zrevrange(key, start, end);
	m.zcard(key);
	m.exec(function (err, res) {
		if (err)
			return self.emit('error', err);
		var ns = res[0];
		if (page && !ns.length)
			return self.emit('nomatch');
		self.emit('begin', res[1]);
		var reader = new Reader(self);
		reader.on('error', self.emit.bind(self, 'error'));
		reader.on('thread', self.emit.bind(self, 'thread'));
		reader.on('post', self.emit.bind(self, 'post'));
		reader.on('endthread', self.emit.bind(self, 'endthread'));
		self._get_each_thread(reader, 0, ns);
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
	reader.get_thread(this.tag, nums[ix], false,
			config.ABBREVIATED_REPLIES || 5);
};

Y.report_error = function (info, ver, callback) {
	var r = this.connect();
	info.time = new Date().getTime();
	r.rpush('error:' + ver, JSON.stringify(info), callback);
};

/* LURKERS */

function Reader(yakusoku) {
	events.EventEmitter.call(this);
	this.y = yakusoku;
}

util.inherits(Reader, events.EventEmitter);
exports.Reader = Reader;

Reader.prototype.get_thread = function (tag, num, redirect_ok, abbrev) {
	var r = this.y.connect();
	var graveyard = (tag == 'graveyard');
	var key = (graveyard ? 'dead:' : 'thread:') + num;
	var self = this;
	r.hgetall(key, function (err, pre_post) {
		if (err)
			return self.emit('error', err);
		if (!graveyard && pre_post.hide)
			return self.emit('nomatch');
		if (_.isEmpty(pre_post)) {
			if (!redirect_ok)
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
		var tags = parse_tags(pre_post.tags);
		if (!graveyard && tags.indexOf(tag) < 0) {
			/* XXX: Should redirect directly to correct thread */
			if (!redirect_ok)
				self.emit('nomatch');
			else
				self.emit('redirect', num, tags[0]);
			return;
		}
		self.emit('begin');
		pre_post.num = num;
		pre_post.time = parseInt(pre_post.time, 10);
		with_body(r, key, pre_post, function (err, op_post) {
			if (err)
				return self.emit('error', err);
			var m = r.multi();
			m.lrange(key + ':posts', -abbrev, -1);
			if (abbrev)
				m.llen(key + ':posts');
			m.exec(function (err, r) {
				if (err)
					return self.emit('error', err);
				var omit = Math.max(r[1] - abbrev, 0);
				extract(op_post);
				self.emit('thread', op_post, omit);
				self._get_each_reply(tag, 0, r[0]);
			});
		});
	});
};

Reader.prototype._get_each_reply = function (tag, ix, nums) {
	if (!nums || ix >= nums.length) {
		this.emit('endthread');
		this.emit('end');
		return;
	}
	var r = this.y.connect();
	var num = parseInt(nums[ix], 10);
	var key = 'post:' + num;
	var next_please = this._get_each_reply.bind(this, tag, ix + 1, nums);
	var self = this;
	r.hgetall(key, function (err, pre_post) {
		if (err)
			return self.emit('error', err);
		if (_.isEmpty(pre_post)
				|| (tag != 'graveyard' && pre_post.hide))
			return next_please();
		pre_post.num = num;
		pre_post.time = parseInt(pre_post.time, 10);
		pre_post.op = parseInt(pre_post.op, 10);
		with_body(r, key, pre_post, function (err, post) {
			if (err)
				return self.emit('error', err);
			extract(post);
			self.emit('post', post);
			next_please();
		});
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
		this.r = redis_client();
		this.r.on('error', console.error.bind(console));
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
			r.hmget(key, ['thumb', 'src'], function (err, rs) {
				if (err)
					cb(err);
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
	if (this.r)
		this.r.quit();
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
		return callback("Thread not found.");
	var self = this;
	fs.readFile('client/fun.js', 'UTF-8', function (err, funJs) {
		if (err)
			return callback(err);
		cache.funThread = op;
		var m = self.connect().multi();
		self._log(m, op, common.EXECUTE_JS, [op, funJs]);
		m.exec(callback);
	});
};

/* HELPERS */

function extract(post) {
	hooks.trigger('extractPost', post);
}

function inline(dest, src) {
	hooks.trigger('inlinePost', {dest: dest, src: src});
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

function tag_key(tag) {
	return tag.length + ':' + tag;
}

function parse_tags(input) {
	if (!input)
		return ['moe'];
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
