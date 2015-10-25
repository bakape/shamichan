/*
Core database initiation and connection
 */

const async = require('async'),
	config = require('../config'),
	redisDB = require('redis'),
	r = require('rethinkdb')

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
function init(cb) {
	async.waterfall([
		next =>
			r.connect({
				host: config.rethink_host,
				port: config.rethinkdb_port
			}, next),
		(conn, next) => {
			rcon = global.rcon = conn

			// Check if database exists
			r.dbList().contains('meguca').run(rcon, next)
		},
		(exists, next) => {
			if (exists)
				return next(null, null)
			initDB(next)
		},
		(res, next) => {
			rcon.use('meguca')
			r.table('main').get('info').run(rcon, next)
		},
		// Intialize main table or check version
		(info, next) => {
			if (info) {
				verifyVersion(info.dbVersion, 'RethinkDB')
				return next(null, null)
			}
			r.table('main').insert({id: 'info', dbVersion}).run(rcon, next)
		},
		// Check redis version
		(res, next) =>
			redis.get('dbVersion', next),
		(version, next) => {
			if (version) {
				verifyVersion(parseInt(version), 'Redis')
				return next(null, null)
			}
			redis.set('dbVersion', dbVersion, next)
		}
		// Pass connection to callback
	], err => cb(err, rcon))
}
exports.init = init

function initDB(cb) {
	async.waterfall([
		next =>
			r.dbCreate('meguca').run(rcon, next),
		(res, next) => {
			rcon.use('meguca')
			r.expr(['main', 'threads', 'replies'])
				.forEach(name => r.tableCreate(name))
				.run(rcon, next)
		},
		// Create secondary indexes for faster queries
		(res, next) => {
			const indexes = [
				['replies', 'op'],
				['threads', 'board'],
				['threads', 'time'],
				['threads', 'bumpTime']
			]
			async.forEach(indexes, ([table, index], cb) =>
				r.table(table).indexCreate(index).run(rcon, cb), next)
		}
	], cb)
}

function verifyVersion(version, dbms) {
	if (version !== dbVersion) {
		throw new Error(`Incompatible ${dbms} database version: ${version} `
			+ 'See docs/migration.md')
	}
}
