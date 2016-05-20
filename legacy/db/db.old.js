/*
Main redis controller module
 */

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
	target_key(id) {
		return id === 'live' ? 'board:' + this.board : 'thread:' + id;
	}
	check_throttle(ip, callback) {
		// So we can spam new threads in debug mode
		if (config.DEBUG)
			return callback();
		redis.exists(`ip:${ip}:throttle:thread`, (err, exists) => {
			if (err)
				return callback(err);
			callback(exists && Muggle('Too soon.'));
		});
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

function postKey(num, op) {
	return `${op == num ? 'thread' : 'post'}:${num}`;
}
