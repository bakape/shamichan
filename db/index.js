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

// Buffers commands, so no need for callback
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

// Establish rethinkDB connection and intialize the database
async function init() {
	rcon = global.rcon = await r.connect({
		host: config.rethink_host,
		port: config.rethinkdb_port
	})
	const isCreated = await r.dbList().contains('meguca').run(rcon)
	if (!isCreated)
		await initDB()
	rcon.use('meguca')
	const info = await r.table('_main').get('info').run(rcon)
	if (info)
		verifyVersion(info.dbVersion, 'RethinkDB')
	else {
		await r.table('_main').insert({
			id: 'info',
			dbVersion,
			post_ctr: 0
		}).run(rcon)
	}
	const redisVersion = await redis.getAsync('dbVersion')
	if (redisVersion)
		verifyVersion(parseInt(redisVersion), 'Redis')
	else
		await redis.setAsync('dbVersion', dbVersion)
	for (let board of config.BOARDS) {
		await initBoard(board)
	}
}
exports.init = init

async function initDB() {
	await r.dbCreate('meguca').run(rcon)
	rcon.use('meguca')
	await r.tableCreate('_main').run(rcon)
}

async function initBoard(board) {
	if (await r.tableList().contains(board).run(rcon))
		return
	await r.tableCreate(board).run(rcon)
	for (let index of ['op', 'time', 'bumptime']) {
		await r.table(board).indexCreate(index).run(rcon)
	}
}

function verifyVersion(version, dbms) {
	if (version !== dbVersion) {
		throw new Error(`Incompatible ${dbms} database version: ${version} `
			+ 'See docs/migration.md')
	}
}
