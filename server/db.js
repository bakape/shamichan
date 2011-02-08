var common = require('./common'),
    config = require('./config'),
    events = require('events'),
    redis = require('redis'),
    util = require('util');

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
		this.r = redis.createClient();
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

Y.kiku = function (thread, callback) {
	if (!this.k) {
		this.k = redis.createClient();
		this.k.on('error', console.error.bind(console));
	}
	this.kikumono = thread;
	function on_subscribe_error(err) {
		deal_with_it.call(this);
		callback(err);
	}
	function on_subscribe(chan, count) {
		deal_with_it.call(this);
		callback(null);
	}
	function deal_with_it() {
		var ev = this.kikumono ? 'subscribe' : 'psubscribe';
		this.k.removeListener(ev, on_subscribe);
		this.k.removeListener('error', on_subscribe_error);
	}
	this.k.on('error', on_subscribe_error.bind(this));
	if (this.kikumono) {
		this.k.on('subscribe', on_subscribe.bind(this));
		this.k.on('message', this._on_message.bind(this, null));
		this.k.subscribe('thread:' + thread);
	}
	else {
		this.k.on('psubscribe', on_subscribe.bind(this));
		this.k.on('pmessage', this._on_message.bind(this));
		this.k.psubscribe('thread:*');
	}
};

Y.kikanai = function (thread) {
	this.k.unsubscribe();
	this.k.removeAllListeners('message');
	this.k.removeAllListeners('pmessage');
};

Y._on_message = function (pat, chan, msg) {
	var num = msg.split(':', 1)[0];
	this.emit('update', chan, parseInt(num), msg.substr(num.length + 1));
};

function is_empty(obj) {
	if (!obj)
		return false;
	for (var key in obj)
		if (obj.hasOwnProperty(key))
			return false;
	return true;
}

Y.insert_post = function (msg, body, ip, update, callback) {
	var r = this.connect();
	var self = this;
	/* Multi isn't needed here, yay. */
	if (msg.op) {
		r.exists('thread:' + msg.op, function (err, exists) {
			if (err)
				callback(err);
			else if (!exists)
				callback('Thread does not exist.');
			else
				self._insert(msg, body, ip, update, callback);
		});
	}
	else
		self._insert(msg, body, ip, update, callback);
};

Y._insert = function (msg, body, ip, update, callback) {
	var r = this.connect();
	var tag_key = 'tag:' + this.tag;
	var self = this;
	r.incr(tag_key + ':ctr', function (err, num) {
		if (err)
			return callback(err);
		var view = {time: msg.time, ip: ip, state: msg.state.join()};
		var op = msg.op;
		if (msg.name)
			view.name = msg.name;
		if (msg.trip)
			view.trip = msg.trip;
		if (msg.email)
			view.email = msg.email;
		if (op)
			view.op = msg.op;
		if (msg.image)
			add_image_view(msg.image, view);

		var key = (op ? 'post:' : 'thread:') + num;
		var bump = !op || view.email != 'sage';
		var m = r.multi();
		if (bump)
			m.incr(tag_key + ':bumpctr');
		m.hmset(key, view);
		m.set(key + ':body', body);
		if (op)
			m.rpush('thread:' + op + ':posts', num);

		/* Need to set client.post here so pubsub doesn't interfere */
		update(num);

		/* Denormalize for backlog */
		view.body = body;
		view.num = num;
		self._log(m, op, num, [common.INSERT_POST, view]);

		m.exec(function (err, results) {
			if (err)
				return callback(err);
			else if (!bump)
				return callback(null, num);
			r.zadd(tag_key + ':threads', results[0], num,
						function (err) {
				if (err)
					callback(err);
				else
					callback(null, num);
			});
		});
	});
};

function add_image_view(image, dest) {
	dest.imgMD5 = image.MD5;
	dest.imgext = image.ext;
	dest.imgsize = image.size;
	dest.imgtime = image.time;
}

Y.add_image = function (post_num, image, callback) {
	var r = this.connect();
	var key = 'post:' + post_num;
	r.exists(key, function (err, exists) {
		if (err)
			callback(err);
		else if (!exists)
			callback("Post does not exist.");
		else {
			var view = {};
			add_image_view(image, view);
			r.hmset(key, view, callback);
		}
	});
};

Y.append_post = function (post, tail, old_state, links, callback) {
	/* TODO: Persist links */
	var m = this.connect().multi();
	var key = (post.op ? 'post:' : 'thread:') + post.num + ':body';
	/* Don't need to check .exists() thanks to client state */
	m.append(key, tail);
	/* XXX: fragile */
	if (old_state[0] != post.state[0] || old_state[1] != post.state[1])
		m.hset(key, 'state', post.state.join());
	var msg = [common.UPDATE_POST, post.num, tail].concat(old_state);
	if (links)
		msg.push(links);
	this._log(m, post.op, post.num, msg);
	m.exec(callback);
};

Y.finish_post = function (post, callback) {
	var m = this.connect().multi();
	var key = (post.op ? 'post:' : 'thread:') + post.num;
	/* Don't need to check .exists() thanks to client state */
	m.hset(key, 'body', post.body);
	m.del(key + ':body');
	m.hdel(key, 'state');
	this._log(m, post.op, post.num, [common.FINISH_POST, post.num]);
	m.exec(callback);
};

Y._log = function (m, op, num, msg) {
	msg = JSON.stringify(msg);
	m.rpush('backlog', msg);
	m.publish('thread:' + (op || num), num + ':' + msg);
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

Y.thread_exists = function (num, callback) {
	this.connect().exists('thread:' + num, callback);
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
	reader.get_thread(nums[ix], false);
};

function Reader(yakusoku) {
	events.EventEmitter.call(this);
	this.y = yakusoku;
}

util.inherits(Reader, events.EventEmitter);
exports.Reader = Reader;

Reader.prototype.get_thread = function (num, redirect_ok) {
	var r = this.y.connect();
	var key = 'thread:' + num;
	var self = this;
	r.hgetall(key, function (err, pre_post) {
		if (err)
			return self.emit('error', err);
		if (is_empty(pre_post)) {
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
		with_body(r, key, pre_post, function (err, post) {
			if (err)
				return self.emit('error', err);
			self.emit('thread', post);
			r.lrange(key + ':posts', 0, -1, function (err, nums) {
				if (err)
					return self.emit('error', err);
				self._get_each_reply(0, nums);
			});
		});
	});
};

Reader.prototype._get_each_reply = function (ix, nums) {
	if (!nums || ix >= nums.length)
		return this.emit('end');
	var r = this.y.connect();
	var num = nums[ix];
	var key = 'post:' + num;
	var next_please = this._get_each_reply.bind(this, ix + 1, nums);
	var self = this;
	r.hgetall(key, function (err, pre_post) {
		if (err)
			return self.emit('error', err);
		if (is_empty(pre_post))
			return next_please();
		pre_post.num = num;
		with_body(r, key, pre_post, function (err, post) {
			if (err)
				return self.emit('error', err);
			var has_next = ix + 1 < nums.length;
			self.emit('post', post, has_next);
			next_please();
		});
	});
};

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
