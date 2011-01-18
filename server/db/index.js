var config = require('../config'),
    events = require('events'),
    redis = require('redis'),
    util = require('util');

var BOARD_TAG = '3:moe';

redis.RedisClient.prototype.wrap = function (f) {
	return (function (e) {
		if (!e) {
			var args = [];
			for (var i = 1; i < arguments.length; i++)
				args.push(arguments[i]);
			f.apply(this, args);
		}
		else
			this.failure(e);
	}).bind(this);
};

redis.RedisClient.prototype.success = function () {
	var args = [null];
	for (var i = 0; i < arguments.length; i++)
		args.push(arguments[i]);
	this._callback.apply(this, args);
	this.quit();
};

redis.RedisClient.prototype.failure = function (e) {
	if (!this.failed) {
		this.failed = true;
		this._callback(e);
		this.quit();
	}
};

function redis_client(callback) {
	var r = redis.createClient();
	r.on('error', callback);
	r._callback = callback;
	return r;
}

function is_empty(obj) {
	if (!obj)
		return false;
	for (var key in obj)
		if (obj.hasOwnProperty(key))
			return false;
	return true;
}

function just_keys(src, keys) {
	var obj = {};
	for (var i = 0; i < keys.length; i++)
		obj[keys[i]] = src[keys[i]];
	return obj;
}

function bump_thread(r, num, callback) {
	var key = 'tag:' + BOARD_TAG;
	r.incr(key + ':bumpctr', r.wrap(function (score) {
		r.zadd(key + ':threads', score, num, r.wrap(callback));
	}));
}

function push_post(r, post, num) {
	r.rpush('post:' + post.op + ':replies', num, r.wrap(function () {
		if (post.email != 'sage')
			bump_thread(r, post.op, r.success.bind(r, num));
		else
			r.success(num);
	}));
}

exports.insert_post = function(msg, body, ip, callback) {
	var r = redis_client(callback);
	/* Multi isn't needed here, yay. */
	r.incr('post:ctr', r.wrap(function (num) {
		var view = {time: msg.time, ip: ip};
		if (msg.name)
			view.name = msg.name;
		if (msg.trip)
			view.trip = msg.trip;
		if (msg.email)
			view.email = msg.email;
		if (msg.op)
			view.op = msg.op;
		if (msg.image)
			add_image_view(msg.image, view);
		var key = 'post:' + num;
		r.hmset(key, view, r.wrap(function () {
			r.set(key + ':body', body, r.wrap(function () {
				if (msg.op)
					return push_post(r, msg, num);
				bump_thread(r, num, r.success.bind(r, num));
			}));
		}));
	}));
};

function add_image_view(image, dest) {
	dest.imgMD5 = image.MD5;
	dest.imgext = image.ext;
	dest.imgsize = image.size;
	dest.imgtime = image.time;
}

exports.add_image = function (post_num, image, callback) {
	var key = 'post:' + post_num;
	var r = redis_client(callback);
	r.exists(key, r.wrap(function (exists) {
		if (!exists)
			r.failure("Post does not exist.");
		else {
			var view = {};
			add_image_view(image, view);
			r.hmset(key, view, r.wrap(r.success.bind(r)));
		}
	}));
};

exports.append_post = function(num, tail, callback) {
	var key = 'post:' + num;
	var r = redis_client(callback);
	r.append(key + ':body', tail, r.wrap(function (new_len) {
		callback(null);
		/* If the post doesn't exist, delete this instead */
		r.exists(key, function (err, exists) {
			if (!err && !exists)
				r.del(key + ':body');
			r.quit();
		});
	}));
};

exports.finish_post = function(num, callback) {
	var key = 'post:' + num;
	var r = redis_client(callback);
	r.get(key + ':body', r.wrap(function (body) {
		r.hmset(key, 'body', body, r.wrap(function () {
			r.del(key + ':body', r.wrap(function () {
				r.success();
			}));
		}));
	}));
};

var Reader = function () {
	events.EventEmitter.call(this);
	this.r = redis_client(this.emit.bind(this, 'error'));
};

util.inherits(Reader, events.EventEmitter);
exports.Reader = Reader;

Reader.prototype.get_tag = function (tag_name) {
	var tag_key = 'tag:' + tag_name.length + ':' + tag_name;
	this.r.zrevrange(tag_key + ':threads', 0, -1,
			this.r.wrap(this._get_each_thread.bind(this, 0)));
};

Reader.prototype._get_each_thread = function (ix, nums) {
	if (ix >= nums.length) {
		this.emit('end');
		this.r.quit();
		delete this.on_thread_end;
		return;
	}
	this.on_thread_end = this._get_each_thread.bind(this, ix + 1, nums);
	this.get_thread(nums[ix]);
};

Reader.prototype.get_thread = function (op) {
	var key = 'post:' + op;
	var self = this;
	this.r.hgetall(key, this.r.wrap(function (pre_post) {
		if (is_empty(pre_post)) {
			if (self.on_thread_end)
				self.on_thread_end();
			else {
				self.emit('end');
				self.r.quit();
			}
			return;
		}
		pre_post.num = op;
		self._with_body(key, pre_post, function (post) {
			self.emit('post', post);
			self.r.lrange(key + ':replies', 0, -1,
					self.r.wrap(self._get_each_reply.bind(self, 0)));
		});
	}));
};

Reader.prototype._get_each_reply = function (ix, nums) {
	if (!nums || ix >= nums.length) {
		if (this.on_thread_end)
			this.on_thread_end();
		else {
			this.emit('end');
			this.r.quit();
		}
		return;
	}
	var num = nums[ix];
	var key = 'post:' + num;
	var next_please = this._get_each_reply.bind(this, ix + 1, nums);
	var self = this;
	this.r.hgetall(key, this.r.wrap(function (pre_post) {
		if (is_empty(pre_post))
			return next_please();
		pre_post.num = num;
		self._with_body(key, pre_post, function (post) {
			self.emit('post', post);
			next_please();
		});
	}));
};

Reader.prototype._with_body = function (key, post, callback) {
	if (post.body !== undefined)
		callback(post);
	else
		this.r.get(key + ':body', this.r.wrap(function (body) {
			if (body !== null) {
				post.body = body;
				post.editing = true;
				callback(post);
				return;
			}
			// Race condition between finishing posts
			this.r.hget(key, 'body', this.r.wrap(function (body) {
				post.body = body;
				callback(post);
			}));
		}));
};
