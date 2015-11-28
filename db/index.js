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
	client.on('error', err =>
		winston.error('Redis error:', err))
	return client
}
const redis = global.redis = redisClient()

/**
 * Establish rethinkDB connection and intialize the database, if needed.
 */
export async function init() {
	rcon = global.rcon = await r.connect({
		host: config.rethink_host,
		port: config.rethinkdb_port
	})
	const isCreated = await r.dbList()
		.contains(config.rethinkdb_database)
		.run(rcon)
	if (!isCreated)
		await initDB()
	else {
		rcon.use(config.rethinkdb_database)
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
	await r.dbCreate(config.rethinkdb_database).run(rcon)
	rcon.use(config.rethinkdb_database)
	await r.tableCreate('main').run(rcon)
	await r.table('main').insert([
		{
			id: 'info',
			dbVersion,
			postCtr: 0
		},
		// History counters of boards. Used for building etags.
		{
			id: 'boardCtrs'
		}
	]).run(rcon)
	await r.tableCreate('threads').run(rcon)
	await r.table('threads').indexCreate('board').run(rcon)
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

/**
 * Assign a handler to a redis subscription channel
 * @param {string} name
 * @param {Function} handler
 */
export function onPublish (name, handler) {
	const redis = redisClient()
	redis.subscribe(name)
	redis.on('message', handler)
}
