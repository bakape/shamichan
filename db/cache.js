/*
Cache of thread and board post parenthood. Kept in redis, for quick
 validations.
 */

const async = require('async'),
	{redis} = global

// Store a post's board and thread parenthood
function cache(m, num, op, board) {
	const part = partition(num)
	m.hset(`boards:${part}`, num, board)
	m.hset(`threads:${part}`, num, op)
}
exports.cache = cache

// Split post numbers into partitions of 1000, for more compact and faster
// storage/lookup
function partition(num) {
	return num - (num % 1000)
}

function uncache(m, num) {
	hashCall(m, 'hdel', num)
}
exports.uncache = uncache

// Muh DRY
function hashCall(m, method, num) {
	const part = partition(num)
	for (let key of ['boards', 'threads']) {
		m[method](`${key}:${part}`, num)
	}
}

function validateOP(num, board, cb) {
	async.waterfall([
		next => {
			const m = redis.multi()
			hashCall(m, 'hget', num)
			m.exec(next)
		},
		(res, next) =>
			next(null, res[0] === board && res[1] === num)
	], cb)
}
exports.validateOP = validateOP
