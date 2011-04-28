var async = require('async'),
    common = require('./common'),
    config = require('./config'),
    events = require('events'),
    redis = require('redis'),
    util = require('util');

var OPs = {};
exports.OPs = OPs;

var subs = {};

function redis_client() {
	return redis.createClient(config.REDIS_PORT || undefined);
}

function Yakusoku() {
	events.EventEmitter.call(this);
	/* TEMP */
	this.tag = '3:moe';
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
		this.r.removeAllListeners();
	}
	if (this.k) {
		this.k.quit();
		this.k.removeAllListeners();
	}
	this.removeAllListeners();
};

function sink_sub(thread, err) {
	console.error(err);
	this.k.quit();
	/* TODO: Inform this.watchers */
	if (subs[thread] == this)
		delete subs[thread];
}

Y.kiku = function (thread, callback) {
	if (thread in subs) {
		subs[thread].watchers.push(this);
		return callback(null);
	}
	var k = redis_client();
	subs[thread] = {k: k, watchers: [this]};
	this.kikumono = thread;
	function on_subscribe_error(err) {
		deal_with_it();
		delete subs[thread];
		callback(err);
	}
	function on_subscribe(chan, count) {
		deal_with_it();
		k.on('message', on_message.bind(sub));
		k.on('error', sink_sub.bind(sub, thread));
		callback(null);
	}
	function deal_with_it() {
		k.removeListener('subscribe', on_subscribe);
		k.removeListener('error', on_subscribe_error);
	}
	k.on('error', on_subscribe_error);
	k.on('subscribe', on_subscribe);
	k.subscribe('thread:' + thread);
};

Y.kikanai = function (thread) {
	this.k.unsubscribe();
	this.k.removeAllListeners('message');
	this.k.removeAllListeners('pmessage');
};

function on_message(chan, msg) {
	var info = msg.split(':', 2);
	var off = info[0].length + info[1].length + 2;
	var num = parseInt(info[0]), kind = parseInt(info[1]);
	this.emit('update', chan, num, kind, msg.substr(off));
}

function on_OP_message(pat, chan, msg) {
	var op = parseInt(chan.match(/thread:(\d+)/)[1]);
	var info = msg.split(':', 2);
	var num = parseInt(info[0]), kind = parseInt(info[1]);
	if (kind == common.INSERT_POST)
		OPs[num] = op;
}

exports.track_OPs = function (callback) {
	var k = redis_client();
	k.psubscribe('thread:*');
	k.on('psubscribe', function () {
		var r = redis_client();
		load_OPs(r, function (err) {
			r.quit();
			callback(err);
		});
	});
	k.on('pmessage', on_OP_message);
};

function load_OPs(r, callback) {
	r.keys('thread:*', function (err, keys) {
		if (err)
			return callback(err);
		async.forEach(keys, function (key, cb) {
			var m = key.match(/^thread:(\d*)(:posts$)?/);
			var op = parseInt(m[1]);
			OPs[op] = op;
			if (!m[2])
				return cb();
			r.lrange(key, 0, -1, function (err, posts) {
				if (err)
					return cb(err);
				for (var i = 0; i < posts.length; i++)
					OPs[parseInt(posts[i])] = op;
				cb();
			});
		}, callback);
	});
}

Y.reserve_post = function (op, callback) {
	this.connect().incr('postctr', function (err, num) {
		if (err)
			return callback(err);
		OPs[num] = op || num;
		callback(null, num);
	});
};

Y.insert_post = function (msg, body, ip, callback) {
	var r = this.connect();
	var tag_key = 'tag:' + this.tag;
	var self = this;
	if (msg.num) {
		if (msg.op && OPs[msg.op] != msg.op) {
			delete OPs[num];
			return callback('Thread does not exist.');
		}
		var view = {time: msg.time, ip: ip, state: msg.state.join()};
		var num = msg.num, op = msg.op;
		if (msg.name)
			view.name = msg.name;
		if (msg.trip)
			view.trip = msg.trip;
		if (msg.email)
			view.email = msg.email;
		if (op)
			view.op = op;

		var key = (op ? 'post:' : 'thread:') + num;
		var bump = !op || view.email != 'sage';
		var m = r.multi();
		if (bump)
			m.hincrby(tag_key, 'bumpctr', 1);
		if (msg.image) {
			if (op)
				m.hincrby('thread:' + op, 'imgctr', 1);
			else
				view.imgctr = 1;
			inline_image(view, msg.image);
		}
		m.hmset(key, view);
		m.set(key + ':body', body);
		if (msg.links)
			m.hmset(key + ':links', msg.links);
		if (op)
			m.rpush('thread:' + op + ':posts', num);
		else
			op = num;

		/* Denormalize for backlog */
		view.body = body;
		view.num = num;
		if (msg.links)
			view.links = msg.links;
		extract_image(view);
		self._log(m, op, num, common.INSERT_POST, [view]);

		m.exec(function (err, results) {
			if (err) {
				delete OPs[num];
				return callback(err);
			}
			if (!bump)
				return callback(null);
			r.zadd(tag_key + ':threads', results[0], op,
						function (err) {
				if (err)
					console.error("Bump error: " + err);
				callback(null);
			});
		});
	}
	else {
		/* TODO: Flatten this conditional once history branch merged */
		callback("No post num.");
	}
};

Y.add_image = function (post, image, callback) {
	var r = this.connect();
	var num = post.num;
	var key = 'post:' + num;
	var self = this;
	r.exists(key, function (err, exists) {
		if (err)
			return callback(err);
		if (!exists)
			return callback("Post does not exist.");
		var m = r.multi();
		self._log(m, post.op, num, common.INSERT_IMAGE, [num, image]);
		m.hmset(key, image);
		m.hincrby('thread:' + post.op, 'imgctr', 1);
		m.exec(callback);
	});
};

Y.append_post = function (post, tail, old_state, links, new_links, callback) {
	var m = this.connect().multi();
	var key = (post.op ? 'post:' : 'thread:') + post.num;
	/* Don't need to check .exists() thanks to client state */
	m.append(key + ':body', tail);
	/* XXX: fragile */
	if (old_state[0] != post.state[0] || old_state[1] != post.state[1])
		m.hset(key, 'state', post.state.join());
	if (new_links && !common.is_empty(new_links))
		m.hmset(key + ':links', new_links);
	var msg = [post.num, tail];
	if (links)
		msg.push(old_state[0], old_state[1], links);
	else if (old_state[1])
		msg.push(old_state[0], old_state[1]);
	else if (old_state[0])
		msg.push(old_state[0]);
	this._log(m, post.op, post.num, common.UPDATE_POST, msg);
	m.exec(callback);
};

Y.finish_post = function (post, callback) {
	var m = this.connect().multi();
	var key = (post.op ? 'post:' : 'thread:') + post.num;
	/* Don't need to check .exists() thanks to client state */
	m.hset(key, 'body', post.body);
	m.del(key + ':body');
	m.hdel(key, 'state');
	this._log(m, post.op, post.num, common.FINISH_POST, [post.num]);
	m.exec(callback);
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
				m.hset(key, 'body', rs[0]);
				m.del(body_key);
				m.hdel(key, 'state');
				var n = parseInt(key.match(/:(\d+)$/)[1]);
				var op = parseInt(rs[1]) || n;
				self._log(m, op, n, common.FINISH_POST, [n]);
				m.exec(cb);
			});
		}, callback);
	});
};

Y._log = function (m, op, num, kind, msg) {
	msg.unshift(kind);
	msg = JSON.stringify(msg);
	console.log("Log:", msg);
	var key = 'thread:' + (op || num);
	m.rpush(key + ':history', msg);
	m.hincrby(key, 'hctr', 1);
	m.publish(key, num + ':' + kind + ':' + msg);
};

Y.fetch_backlog = function (sync, watching, callback) {
	var r = this.connect();
	r.lrange('backlog', sync, -1, function (err, log) {
		if (err)
			return callback(err);
		// TODO: Do something with watching
		// Naive impl for now
		callback(null, sync + log.length, log);
	});
};

Y.get_sync_number = function (callback) {
	this.connect().llen('backlog', callback);
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

Y.get_tag = function () {
	var r = this.connect();
	var self = this;
	r.zrevrange('tag:' + this.tag + ':threads', 0, -1, function (err, ns) {
		if (err)
			return self.emit('error', err);
		self.emit('begin');
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
		reader.removeAllListeners();
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
	reader.get_thread(nums[ix], false, true);
};

Y.report_error = function (info, ver, callback) {
	var r = this.connect();
	info.time = new Date().getTime();
	r.rpush('error:' + ver, JSON.stringify(info), callback);
};

function Reader(yakusoku) {
	events.EventEmitter.call(this);
	this.y = yakusoku;
}

util.inherits(Reader, events.EventEmitter);
exports.Reader = Reader;

Reader.prototype.get_thread = function (num, redirect_ok, abbrev) {
	var r = this.y.connect();
	var key = 'thread:' + num;
	var self = this;
	r.hgetall(key, function (err, pre_post) {
		if (err)
			return self.emit('error', err);
		if (common.is_empty(pre_post)) {
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
		self.emit('begin');
		pre_post.num = num;
		with_body(r, key, pre_post, function (err, op_post) {
			if (err)
				return self.emit('error', err);
			var shonen = abbrev ? -config.ABBREVIATED_REPLIES : 0;
			var m = r.multi();
			m.lrange(key + ':posts', shonen, -1);
			if (abbrev)
				m.llen(key + ':posts');
			m.exec(function (err, r) {
				if (err)
					return self.emit('error', err);
				var omit = Math.max(r[1] + shonen, 0);
				extract_image(op_post);
				self.emit('thread', op_post, omit);
				self._get_each_reply(0, r[0]);
			});
		});
	});
};

Reader.prototype._get_each_reply = function (ix, nums) {
	if (!nums || ix >= nums.length) {
		this.emit('endthread');
		this.emit('end');
		return;
	}
	var r = this.y.connect();
	var num = nums[ix];
	var key = 'post:' + num;
	var next_please = this._get_each_reply.bind(this, ix + 1, nums);
	var self = this;
	r.hgetall(key, function (err, pre_post) {
		if (err)
			return self.emit('error', err);
		if (common.is_empty(pre_post))
			return next_please();
		pre_post.num = num;
		with_body(r, key, pre_post, function (err, post) {
			if (err)
				return self.emit('error', err);
			extract_image(post);
			self.emit('post', post);
			next_please();
		});
	});
};

var image_attrs;
function extract_image(post) {
	if (!image_attrs)
		image_attrs = require('./pix').image_attrs;
	if (!(image_attrs[0] in post))
		return;
	var image = {};
	image_attrs.forEach(function (key) {
		image[key] = post[key];
		delete post[key];
	});
	if (image.dims.split)
		image.dims = image.dims.split(',');
	image.size = parseInt(image.size);
	post.image = image;
}

function inline_image(post, image) {
	if (!image_attrs)
		image_attrs = require('./pix').image_attrs;
	image_attrs.forEach(function (key) {
		post[key] = image[key];
	});
}

function with_body(r, key, post, callback) {
	/* Convenience */
	post.time = parseInt(post.time);
	post.op = parseInt(post.op);

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
