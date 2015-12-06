/*
Cache of thread and board post parenthood. Kept in redis, for quick
 validations.
 */

const {redis} = global

/**
 * Store a new post's board and thread parenthood
 * @param {redis.multi} m
 * @param {int} id
 * @param {int} op
 * @param {string} board
 */
export function add(m, id, op, board) {
	// Add to set of currently open posts
	m.sadd('liveposts', id)

	const part = partition(id)
	m.hset(`boards:${part}`, id, board)
	m.hset(`threads:${part}`, id, op)
}

/**
 * Split post numbers into partitions of 1000, for more compact and faster
 * storage/lookup
 * @param {int} id
 * @returns {int}
 */
function partition(id) {
	return id - (id % 1000)
}

/**
 * Remove post parenthood from cache
 * @param {redis.multi} m
 * @param {int} id
 */
export function remove(m, id) {
	hashCall(m, 'hdel', id)
}

/**
 * Helper function for more DRY-ness
 * @param {redis.multi} m
 * @param {string} method
 * @param {int} id
 */
function hashCall(m, method, id) {
	const part = partition(id)
	for (let key of ['boards', 'threads']) {
		m[method](`${key}:${part}`, id)
	}
}

/**
 * Confirm the specified thread exists on specific board
 * @param {int} id
 * @param {string} board
 * @returns {boolean}
 */
export async function validateOP(id, board) {
	const m = redis.multi()
	getParenthood(m, id)
	const res = await m.execAsync()
	return res[0] === board && res[1] == id
}

/**
 * Return parent thread of post. Can also be used to ensure post exists
 * @param {int} id
 * @returns {(int|NaN)}
 */
export async function parentThread(id) {
    return parseInt(await redis.hgetAsync(`threads:${partition(id)}`, id))
}

/**
 * Get the parent board and thread of a post
 * @param {redis.multi} m
 * @param {int} id
 */
export function getParenthood(m, id) {
	hashCall(m, 'hget', id)
}
