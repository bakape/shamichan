/*
Not yet ported and in line for heavy refactor
 */

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

/* END BOILERPLATE CITY */

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
