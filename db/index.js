/*
Core database initiation and connection
 */

const config = require('../config'),
	Promise = require('bluebird'),
	redisDB = require('redis'),
	r = require('rethinkdb'),
	util = require('./util')

// Convert callback style to promise style for use with ES7 async functions
Promise.promisifyAll(redisDB.RedisClient.prototype)
Promise.promisifyAll(redisDB.Multi.prototype)

const dbVersion = 2
let rcon

/**
 * Creates redis client. Buffers commands, so no need for callback.
 * @returns {redis}
 */
export function redisClient() {
	const client = redisDB.createClient({
		host: config.redis_host,
		port: config.REDIS_PORT
	})
	client.select(config.redis_database || 0)
	return client
}
const redis = global.redis = redisClient()
redis.on('error', err => winston.error('Redis error:', err))

/**
 * Establish rethinkDB connection and intialize the database, if needed.
 */
export async function init() {
	rcon = global.rcon = await r.connect({
		host: config.rethink_host,
		port: config.rethinkdb_port
	})
	const isCreated = await r.dbList().contains('meguca').run(rcon)
	if (!isCreated)
		await initDB()
	else {
		rcon.use('meguca')
		const info = await r.table('main').get('info').run(rcon)
		verifyVersion(info.dbVersion, 'RethinkDB')
	}
	const redisVersion = await redis.getAsync('dbVersion')
	if (redisVersion)
		verifyVersion(parseInt(redisVersion), 'Redis')
	else
		await redis.setAsync('dbVersion', dbVersion)
}

/**
 * Create needed tables and initial documents
 */
async function initDB() {
	await r.dbCreate('meguca').run(rcon)
	rcon.use('meguca')
	await r.tableCreate('main').run(rcon)
	await r.table('main').insert([
		{
			id: 'info',
			dbVersion,
			postCtr: 0
		},
		// History counters of booards. Used for building e-tags.
		{
			id: 'boardCtrs'
		}
	]).run(rcon)
	await r.tableCreate('threads').run(rcon)
	for (let index of ['time', 'bumptime', 'board']) {
		await r.table('threads').indexCreate(index).run(rcon)
	}

	// Index by reply count
	await r.table('threads').indexCreate('replyCount', util.countReplies)
}

/**
 * Verify the version of the database is compatible or throw error
 * @param {int} version
 * @param {string} dbms - Name of DBMS
 */
function verifyVersion(version, dbms) {
	if (version !== dbVersion) {
		throw new Error(`Incompatible ${dbms} database version: ${version} `
			+ 'See docs/migration.md')
	}
}
