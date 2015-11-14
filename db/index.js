/*
Core database initiation and connection
 */

const bluebird = require('bluebird'),
	config = require('../config'),
	redisDB = require('redis'),
	r = require('rethinkdb')

// Convert callback style to promise style for use with ES7 async functions
bluebird.promisifyAll(redisDB.RedisClient.prototype)
bluebird.promisifyAll(redisDB.Multi.prototype)

const dbVersion = 2
let rcon

/**
 * Creates redis client. Buffers commands, so no need for callback.
 * @returns {redis}
 */
function redisClient() {
	const client = redisDB.createClient({
		host: config.redis_host,
		port: config.REDIS_PORT
	})
	client.select(config.redis_database || 0)
	return client
}
exports.redisClient = redisClient
const redis = global.redis = redisClient()
redis.on('error', err => winston.error('Redis error:', err))

/**
 * Establish rethinkDB connection and intialize the database, if needed.
 */
async function init() {
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
exports.init = init

/**
 * Create needed tables and initial documents
 */
async function initDB() {
	await r.dbCreate('meguca').run(rcon)
	rcon.use('meguca')
	await r.tableCreate('main').run(rcon)
	await r.table('main').insert({
		id: 'info',
		dbVersion,
		post_ctr: 0
	}).run(rcon)
	await r.tableCreate('posts').run(rcon)
	for (let index of ['op', 'time', 'bumptime', 'board']) {
		await r.table('posts').indexCreate(index).run(rcon)
	}
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
