var config = require('../config'),
    events = require('events'),
    redis = require('redis'),
    util = require('util');

function Yakusoku() {
	events.EventEmitter.call(this);
}

util.inherits(Yakusoku, events.EventEmitter);
exports.Yakusoku = Yakusoku;
var Y = Yakusoku.prototype;

Y.connect = function () {
	if (!this.r) {
		this.r = redis.createClient();
		this.r.on('error', (function (err) {
			console.error(err);
			delete this.r;
		}).bind(this));
		/* TEMP */
		this.tag = '3:moe';
	}
	return this.r;
};

Y.disconnect = function () {
	if (this.r)
		this.r.quit();
};

function is_empty(obj) {
	if (!obj)
		return false;
	for (var key in obj)
		if (obj.hasOwnProperty(key))
			return false;
	return true;
}

Y.insert_post = function (msg, body, ip, callback) {
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
				self._insert(msg, body, ip, callback);
		});
	}
	else
		self._insert(msg, body, ip, callback);
};

Y._insert = function (msg, body, ip, callback) {
	var r = this.connect();
	r.incr('tag:' + this.tag + ':ctr', function (err, num) {
		if (err)
			return callback(err);
		var view = {time: msg.time, ip: ip};
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
		var tag_key = 'tag:' + this.tag;
		var bump = !op || view.email != 'sage';
		var m = r.multi();
		if (bump)
			m.incr(tag_key + ':bumpctr');
		m.hmset(key, view);
		m.set(key + ':body', body);
		if (op)
			m.rpush('thread:' + op + ':posts', num);

		view.body = body;
		m.rpush('backlog', '+' + JSON.stringify(view));
		delete view.body;

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

Y.append_post = function (num, is_reply, tail, limit, callback) {
	var r = this.connect();
	var key = (is_reply ? 'post:' : 'thread:') + num + ':body';
	/* Don't need to check .exists() thanks to client state */
	r.append(key, tail, function (err, new_len) {
		if (err)
			callback(err);
		else if (new_len > limit)
			trim_post(r, key, limit, callback, 0);
		else
			callback(null, new_len);
	});
};

function trim_post(r, key, limit, callback, tries) {
	r.watch(key);
	r.substr(key, 0, limit, function (err, body) {
		if (err || tries > 5) {
			console.error("Warning: Overlong post permitted");
			return callback(err);
		}
		r.multi().set(key, body).exec(function (err) {
			if (err)
				trim_post(r, key, limit, callback, tries+1);
			else
				callback(null, limit);
		});
	});
}

Y.finish_post = function (num, is_reply, callback) {
	var r = this.connect();
	var key = (is_reply ? 'post:' : 'thread:') + num;
	/* Don't need to check .exists() thanks to client state */
	r.get(key + ':body', function (err, body) {
		if (err)
			return callback(err);
		r.hmset(key, 'body', body, function (err) {
			if (err)
				callback(err);
			else
				r.del(key + ':body', callback);
		});
	});
};

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
		return;
	}
	var next_please = this._get_each_thread.bind(this, reader, ix+1, nums);
	reader.once('end', next_please);
	reader.once('nomatch', next_please);
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
